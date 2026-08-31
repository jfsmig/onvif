// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package networking

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

// DefaultTimeout bounds a single SOAP exchange on the client NewClient builds for itself.
// It is a backstop, not the primary mechanism: cancellation and deadlines normally arrive
// through the context passed to CallMethod. It matters when a caller passes a context with
// no deadline, which would otherwise leave a request with nothing to stop it.
const DefaultTimeout = 30 * time.Second

// Xlmns XML Schema
var Xlmns = map[string]string{
	"onvif":   "http://www.onvif.org/ver10/schema",
	"tds":     "http://www.onvif.org/ver10/device/wsdl",
	"trt":     "http://www.onvif.org/ver10/media/wsdl",
	"tev":     "http://www.onvif.org/ver10/events/wsdl",
	"tptz":    "http://www.onvif.org/ver20/ptz/wsdl",
	"timg":    "http://www.onvif.org/ver20/imaging/wsdl",
	"tan":     "http://www.onvif.org/ver20/analytics/wsdl",
	"xmime":   "http://www.w3.org/2005/05/xmlmime",
	"wsnt":    "http://docs.oasis-open.org/wsn/b-2",
	"xop":     "http://www.w3.org/2004/08/xop/include",
	"wsa":     "http://www.w3.org/2005/08/addressing",
	"wstop":   "http://docs.oasis-open.org/wsn/t-1",
	"wsntw":   "http://docs.oasis-open.org/wsn/bw-2",
	"wsrf-rw": "http://docs.oasis-open.org/wsrf/rw-2",
	"wsaw":    "http://www.w3.org/2006/05/addressing/wsdl",
}

// Client for a new device of onvif and DeviceInfo
// struct represents an abstract ONVIF device.
// It contains methods, which helps to communicate with ONVIF device
type Client struct {
	xaddr    string
	username string
	password string

	// Discovered with the WS-discovery Probematch
	uuid string

	httpClient *http.Client

	// Discovered with the WS-discovery ProbeMatch
	endpoints map[string]string
}

type ClientAuth struct {
	Username string
	Password string
}

type ClientInfo struct {
	Xaddr string
	Uuid  string
}

// NewClient constructs an ONVIF Client entity.
//
// The client used is guaranteed to refuse HTTP redirects, because a 307 or 308 would replay
// the credential-bearing SOAP body at the redirect target (see refuseRedirect). If the
// supplied httpClient has no CheckRedirect of its own, a shallow copy carrying the policy is
// stored rather than mutating the caller's struct, which would be a surprising side effect
// and a data race if they share it. The copy keeps the same Transport, so connection pooling
// and any Timeout are preserved. A caller who has set CheckRedirect is left alone.
func NewClient(ref ClientInfo, httpClient *http.Client) (*Client, error) {
	dev := &Client{
		xaddr:      ref.Xaddr,
		uuid:       ref.Uuid,
		username:   "",
		password:   "",
		httpClient: httpClient,
		endpoints:  make(map[string]string),
	}

	dev.AddEndpoint("Device", "http://"+dev.xaddr+"/onvif/device_service")
	switch {
	case dev.httpClient == nil:
		dev.httpClient = &http.Client{CheckRedirect: refuseRedirect, Timeout: DefaultTimeout}
	case dev.httpClient.CheckRedirect == nil:
		clone := *dev.httpClient
		clone.CheckRedirect = refuseRedirect
		dev.httpClient = &clone
	}

	return dev, nil
}

func (client *Client) GetUUID() string { return client.uuid }

func (client *Client) SetUUID(uuid string) { client.uuid = uuid }

func (client *Client) SetAuth(auth ClientAuth) {
	client.username = auth.Username
	client.password = auth.Password
}

func (client *Client) GetAuth() ClientAuth { return ClientAuth{client.username, client.password} }

// GetServices return available endpoints
func (client *Client) GetServices() map[string]string { return client.endpoints }

// GetEndpoint returns specific ONVIF service endpoint address
func (client *Client) GetEndpoint(name string) string { return client.endpoints[name] }

func (client *Client) AddEndpoint(Key, Value string) {
	//use lowCaseKey
	//make key having ability to handle Mixed Case for Different vendor devcie (e.g. Events EVENTS, events)
	lowCaseKey := strings.ToLower(Key)

	// Replace host with host from device params.
	if u, err := url.Parse(Value); err == nil {
		u.Host = client.xaddr
		Value = u.String()
	}

	client.endpoints[lowCaseKey] = Value
}

// CallMethod functions call a method, defined <method> struct.
// You should use Authenticate method to call authorized requests.
func (client *Client) CallMethod(ctx context.Context, method interface{}) (*http.Response, error) {
	pkgPath := strings.Split(reflect.TypeOf(method).PkgPath(), "/")
	pkg := strings.ToLower(pkgPath[len(pkgPath)-1])

	endpoint, err := client.getEndpoint(pkg)
	if err != nil {
		return nil, err
	}

	output, err := xml.MarshalIndent(method, "  ", "    ")
	if err != nil {
		return nil, err
	}

	soap, err := buildMethodSOAP(string(output))
	if err != nil {
		return nil, err
	}

	soap.AddRootNamespaces(Xlmns)

	//Auth Handling
	if client.username != "" && client.password != "" {
		soap.AddWSSecurity(client.username, client.password)
	}

	return SendSoap(ctx, client.httpClient, endpoint, soap.String())
}

// getEndpoint functions get the target service endpoint in a better way
func (client *Client) getEndpoint(endpoint string) (string, error) {

	// common condition, endpointMark in map we use this.
	if endpointURL, bFound := client.endpoints[endpoint]; bFound {
		return endpointURL, nil
	}

	//but ,if we have endpoint like event、analytic
	//and sametime the Targetkey like : events、analytics
	//we use fuzzy way to find the best match url
	var endpointURL string
	for targetKey := range client.endpoints {
		if strings.Contains(targetKey, endpoint) {
			endpointURL = client.endpoints[targetKey]
			return endpointURL, nil
		}
	}
	return endpointURL, errors.New("target endpoint service not found")
}

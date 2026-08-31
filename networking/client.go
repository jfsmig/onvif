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
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/jfsmig/onvif/utils"
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

// serviceEndpointKeys names, per Go package, the endpoint keys that denote that service.
//
// Only `event` needs more than its own name: the package is event/ because that is what the
// ONVIF WSDL calls the port type, while GetCapabilities reports the endpoint under `Events`,
// which AddEndpoint lowercases.
var serviceEndpointKeys = map[string][]string{
	"event": {"events", "event"},
}

// knownServiceKeys is every endpoint key sdk.load can extract from a GetCapabilities reply:
// the children of Capabilities, then those of Capabilities/Extension. Each names a distinct
// ONVIF service, which is what makes them ineligible for the substring fallback below.
var knownServiceKeys = map[string]bool{
	"device": true, "media": true, "ptz": true, "events": true, "imaging": true,
	"analytics": true, "deviceio": true, "display": true, "recording": true,
	"search": true, "replay": true, "receiver": true, "analyticsdevice": true,
}

// HasEndpoint resolves a service name to the endpoint the device advertised for it, and
// reports whether one was found.
//
// The name is the one CallMethod derives from the request struct's package, so a caller can
// ask in advance whether a service is reachable and get the same answer the call would.
// ONVIF makes whole services conditional, and this is how a caller tests for one without
// sending anything.
//
// Resolution is: the exact key, then the aliases in serviceEndpointKeys, then — only as a
// concession to vendors who invent key names — a key that contains the service name and is
// not itself a known ONVIF service.
//
// That last exclusion is the point. The previous implementation accepted any containing key,
// so `analytics` resolved to `analyticsdevice` on a device advertising the latter alone:
// two different services, one silently answering for the other. It also returned the first
// match from a map range, so with several candidates the answer varied between runs.
func (client *Client) HasEndpoint(name string) (string, bool) {
	for _, key := range append([]string{name}, serviceEndpointKeys[name]...) {
		if endpointURL, found := client.endpoints[key]; found {
			return endpointURL, true
		}
	}

	// Deterministic: shortest, then lexicographic, so a device advertising several odd keys
	// resolves the same way every time.
	best := ""
	for key := range client.endpoints {
		if knownServiceKeys[key] || !strings.Contains(key, name) {
			continue
		}
		if best == "" || len(key) < len(best) || (len(key) == len(best) && key < best) {
			best = key
		}
	}
	if best == "" {
		return "", false
	}
	return client.endpoints[best], true
}

// getEndpoint returns the endpoint for a service, or ErrNoService naming it.
func (client *Client) getEndpoint(endpoint string) (string, error) {
	endpointURL, found := client.HasEndpoint(endpoint)
	if !found {
		return "", fmt.Errorf("%w: %s", utils.ErrNoService, endpoint)
	}
	return endpointURL, nil
}

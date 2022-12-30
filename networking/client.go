package networking

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

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

// NewClient function construct a ONVIF Client entity
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
	if dev.httpClient == nil {
		dev.httpClient = &http.Client{}
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
func (client *Client) CallMethod(method interface{}) (*http.Response, error) {
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
	soap.AddAction()

	//Auth Handling
	if client.username != "" && client.password != "" {
		soap.AddWSSecurity(client.username, client.password)
	}

	return SendSoap(client.httpClient, endpoint, soap.String())
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

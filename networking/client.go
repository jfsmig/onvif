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
	"maps"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jfsmig/onvif/utils"
)

// DefaultTimeout bounds a single SOAP exchange on the client NewClient builds for itself.
// It is a backstop, not the primary mechanism: cancellation and deadlines normally arrive
// through the context passed to CallMethod. It matters when a caller passes a context with
// no deadline, which would otherwise leave a request with nothing to stop it.
const DefaultTimeout = 30 * time.Second

// Xlmns is every namespace prefix declared on the envelope root.
//
// It has to cover the prefixes a request's *character data* can carry and not only the ones
// its element names use, which is why two of these look redundant:
//
//   - tns1 is the ONVIF topic namespace. A ConcreteSet topic expression is a QName in
//     character data -- ONVIF Core section 9.6.3 gives the grammar as "RootTopic ::= QName"
//     and requires the prefix to "correspond to a valid Topic Namespace definition" -- so a
//     filter reading "tns1:RuleEngine//." needs it bound, and section 9.10.3's own envelope
//     declares exactly this. Without it a device answers InvalidTopicExpressionFault, which
//     reads as a camera that does not support the topic rather than a client that did not
//     declare it.
//   - tt is the same URI as onvif, deliberately. Every ONVIF example spells the schema
//     namespace tt: (sections 9.4.1, 9.10.6), and an ElementItem's value is captured as raw
//     XML and written back verbatim -- so a rule read from a camera and modified carries the
//     device's own tt: prefix into an envelope this library builds. Two prefixes for one
//     namespace is legal, and the alternative is an envelope that is not namespace-well-formed
//     and that a device rejects whole.
var Xlmns = map[string]string{
	"onvif":   "http://www.onvif.org/ver10/schema",
	"tt":      "http://www.onvif.org/ver10/schema",
	"tns1":    "http://www.onvif.org/ver10/topics",
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

// Client is one connection to one ONVIF device: its identity, its credentials and the
// service endpoints it advertises.
//
// Ownership rule, which the sdk fan-outs depend on and nothing else enforces: xaddr,
// username, password, uuid and endpoints are written during construction and by
// sdk.load(), both of which complete before the Appliance is handed to a caller, so that
// return orders every write against every later read. AddEndpoint, SetAuth and SetUUID are
// construction-time setters -- calling one while Fetch* calls are in flight is a data race.
// clockOffset is the exception and is atomic; see its comment.
type Client struct {
	xaddr    string
	username string
	password string

	// Discovered with the WS-discovery Probematch
	uuid string

	httpClient *http.Client

	// Discovered with the WS-discovery ProbeMatch
	endpoints map[string]string

	// clockOffset is deviceUTC - localUTC, learnt once at load time and applied to every
	// UsernameToken Created stamp. An offset rather than an absolute instant, because the
	// local clock keeps advancing: a constant offset stays correct for the whole life of a
	// long-lived client, which a stored device timestamp would not.
	//
	// Atomic, unlike the fields above. Those are written during construction and by
	// sdk.load(), both of which complete before the Appliance is handed to a caller, so
	// the return itself orders them against every later read. This one has an exported
	// setter and is read by every concurrent CallMethod, so it is the one field where that
	// argument does not hold. An atomic costs nothing on this path and settles it.
	clockOffset atomic.Int64
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

// SetClockOffset records deviceUTC - localUTC for this device.
//
// sdk.WrapClient computes it from the GetSystemDateAndTime it already issues as its
// liveness probe. A caller driving this package directly can set it by hand; leaving it at
// zero reproduces the behaviour of stamping in local time.
func (client *Client) SetClockOffset(offset time.Duration) {
	client.clockOffset.Store(int64(offset))
}

// ClockOffset returns the offset SetClockOffset recorded, zero if none was.
func (client *Client) ClockOffset() time.Duration {
	return time.Duration(client.clockOffset.Load())
}

// deviceNow is the current instant as the device sees it.
//
// The Location stays UTC deliberately: AddWSSecurityAt formats Created with
// time.RFC3339Nano, so a non-UTC Location would emit "+02:00" where a device expects "Z".
func (client *Client) deviceNow() time.Time {
	return time.Now().UTC().Add(client.ClockOffset())
}

// GetServices returns a copy of the endpoints the device advertised.
//
// A copy, because the map is the client's own routing table: CallMethod resolves against
// it, and bin/onvif-cli/dump.go hands this result straight to a json.Encoder, so a caller
// that adjusted what it had just printed would silently reroute the client's calls.
func (client *Client) GetServices() map[string]string { return maps.Clone(client.endpoints) }

// GetEndpoint returns specific ONVIF service endpoint address
func (client *Client) GetEndpoint(name string) string { return client.endpoints[name] }

func (client *Client) AddEndpoint(Key, Value string) {
	//use lowCaseKey
	//make key having ability to handle Mixed Case for Different vendor devcie (e.g. Events EVENTS, events)
	lowCaseKey := strings.ToLower(Key)

	// Replace host with host from device params.
	if u, err := url.Parse(Value); err == nil {
		u.Host = client.xaddr
		// And drop any account the device embedded in the address it advertised. We
		// authenticate with a WS-Security UsernameToken of our own, so a credential in the
		// URI is never wanted -- and net/http turns req.URL.User into an HTTP Basic
		// Authorization header, so keeping it would put a device-chosen credential on every
		// request and into any error or log line carrying the endpoint. Same rule as
		// FetchStreamURI's: a secret must not reach a log or a dump.
		u.User = nil
		Value = u.String()
	}

	client.endpoints[lowCaseKey] = Value
}

// AtDeviceHost re-points a URI the device handed out at the host it is actually reachable
// at, keeping the port, the path and the query.
//
// Same reason as AddEndpoint's rewrite of an advertised XAddr above -- a device fills those
// in from its own point of view, so one behind a port mapping, or one whose lease has moved,
// advertises an address that does not resolve from here; ONVIF Core section 9.10.4 shows a
// subscription reference advertised exactly that way, naming 160.10.64.10.
//
// The port is where this parts company with AddEndpoint, which replaces host and port
// together. A pull-point subscription commonly answers on a port of its own -- the reply in
// event/namespace_test.go has the device on :80 and the subscription on :8000 -- so blindly
// taking ours would send every pull to the device service. Blindly keeping the device's is
// no better: a camera reached through a port map advertises its internal port, which does
// not resolve from here either. The two cases are told apart by the host the device named:
//
//   - it named the host we already reach it at, so it is describing a port on the machine we
//     are talking to and that port is real. Keep it.
//   - it named a different host, so it is describing itself from a vantage point we do not
//     share -- and a port from that vantage point is no more usable than the host was. Take
//     ours, both halves.
//
// A URI that will not parse comes back unchanged: it is the device's own answer, and refusing
// it here would replace a request that might work with an error that cannot.
func (client *Client) AtDeviceHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	// Drop any account the device embedded, for the reason AddEndpoint drops it.
	u.User = nil

	host := client.xaddr
	if port := u.Port(); port != "" && u.Hostname() == hostnameOf(client.xaddr) {
		// net/url has no setter for the host part alone, and Host carries both, so it is
		// rebuilt -- JoinHostPort, because an IPv6 literal needs its brackets back.
		host = net.JoinHostPort(u.Hostname(), port)
	}
	u.Host = host

	if u.Scheme == "" {
		// A relative reference -- some devices answer "/onvif/Subscription?Idx=0" -- would
		// otherwise serialise as "//host/onvif/..." and lose its scheme.
		u.Scheme = "http"
	}
	return u.String()
}

// hostnameOf is the host half of an authority, brackets already stripped from an IPv6
// literal, and the whole string when it carries no port.
func hostnameOf(authority string) string {
	if host, _, err := net.SplitHostPort(authority); err == nil {
		return host
	}
	return strings.Trim(authority, "[]")
}

// WSAActor is implemented by a request struct that must travel with a wsa:Action header.
//
// The action string belongs to the operation, so it lives next to the request types rather
// than in this package or in calls.txt: keeping it out of calls.txt keeps that file a pure
// list of names, leaves the generated wrappers untouched, and keeps the generator's 1:1
// invariant out of the question entirely. See event/actions.go.
type WSAActor interface {
	// WSAAction returns the wsaw:Action of the operation, as the service WSDL declares it.
	WSAAction() string
}

// WSAAddressee is implemented by a request struct that carries its own destination.
//
// ONVIF Core section 9.1 gives a pull-point subscription a subscription manager of its own:
// the URI arrives in CreatePullPointSubscriptionResponse, and the operations of the
// PullPointSubscription and bw-2 SubscriptionManager port types are POSTed there rather than
// to the event service, with that URI repeated in a wsa:To header. Compare the requests in
// ONVIF Core sections 9.10.5 and 9.10.7, which carry both wsa:Action and wsa:To, with the
// one in section 9.10.3, which carries neither.
//
// Nothing the device advertised names that URI, so the endpoint map cannot hold it and
// package-name routing cannot reach it: the request itself is the only thing that knows
// where it goes. Opt-in per request type, like WSAActor above, so the 200 operations
// addressed by service go out byte-identical to before.
//
// An empty result means "route on the package name", so the zero value of every such
// request struct stays usable.
//
// The method must have a value receiver. CallMethod is handed an interface holding a value
// -- the generated wrappers pass their request by value -- so a pointer receiver would not
// satisfy this assertion and the request would silently fall back to package routing.
type WSAAddressee interface {
	// WSATo returns the URI this request is addressed to, or "" to route on the package.
	WSATo() string
}

// CallMethod functions call a method, defined <method> struct.
// You should use Authenticate method to call authorized requests.
func (client *Client) CallMethod(ctx context.Context, method interface{}) (*http.Response, error) {
	endpoint, err := client.methodEndpoint(method)
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

	// WS-Addressing, for the services that require it and only those.
	//
	// ONVIF requires WS-Addressing for the event service, whose operations are
	// WS-BaseNotification ones; the device, media and PTZ examples in the specification
	// carry no addressing header at all. So the action is opt-in per request type -- a
	// request struct declares one by implementing WSAActor -- rather than emitted for all
	// 205 operations, where it would be unrequested risk against devices that work today.
	//
	// gosoap deliberately does not mark the header mustUnderstand, so a device that
	// ignores WS-Addressing keeps working.
	if actor, ok := method.(WSAActor); ok {
		soap.AddAction(actor.WSAAction())
	}

	// wsa:To, for a request addressed to a subscription manager rather than to a service.
	//
	// Emitted only when the request named a destination, so it never appears on one routed
	// by package name -- section 9.10.3's CreatePullPointSubscription carries no wsa:To, and
	// adding one there would be a header ONVIF's own example does not have.
	//
	// Not marked mustUnderstand, like wsa:Action: most devices route on the POST URL alone
	// and ignore WS-Addressing, and they must keep working.
	if addressee, ok := method.(WSAAddressee); ok {
		addWSATo(soap, addressee.WSATo())
	}

	// Auth handling.
	//
	// Stamped in device time, not local time. A device compares Created against its own
	// clock and rejects a token more than a few seconds out -- which is why gosoap exposes
	// AddWSSecurityAt at all -- and camera clocks drift. Stamping locally meant that on a
	// skewed camera the unauthenticated probe succeeded and then every real call failed
	// 401, which is the most confusing failure this library can produce. With no offset
	// learnt, deviceNow() is time.Now().UTC() and the header is byte-identical to before.
	//
	// The guard is on the username alone. Requiring a non-empty password too meant an
	// account with an empty password got no wsse:Security header at all, rather than a
	// token carrying an empty password. An empty ClientAuth still sends nothing.
	if client.username != "" {
		soap.AddWSSecurityAt(client.username, client.password, client.deviceNow())
	}

	return SendSoap(ctx, client.httpClient, endpoint, soap.String())
}

// methodEndpoint decides where a request is POSTed: the destination the request names, else
// the endpoint the device advertised for the service its package routes to.
//
// The destination is consulted first on purpose. A pull point lives at a URI the device
// handed out, so a camera whose GetCapabilities omits the event service would otherwise
// fail with ErrNoService on a subscription it had just granted.
func (client *Client) methodEndpoint(method interface{}) (string, error) {
	if addressee, ok := method.(WSAAddressee); ok {
		if to := addressee.WSATo(); to != "" {
			return to, nil
		}
	}

	pkgPath := strings.Split(reflect.TypeOf(method).PkgPath(), "/")
	return client.getEndpoint(strings.ToLower(pkgPath[len(pkgPath)-1]))
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

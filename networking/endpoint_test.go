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
	"errors"
	"strings"
	"testing"

	"github.com/jfsmig/onvif/v2/utils"
)

// clientWithEndpoints builds a Client holding exactly the given endpoint keys, bypassing
// NewClient so that the locally seeded "device" entry does not perturb the cases.
func clientWithEndpoints(keys ...string) *Client {
	client := &Client{xaddr: "192.0.2.1:80", endpoints: make(map[string]string)}
	for _, key := range keys {
		client.endpoints[key] = "http://192.0.2.1/onvif/" + key
	}
	return client
}

// TestHasEndpointResolution covers the three ways a service name reaches an endpoint, and
// the case where it must not.
func TestHasEndpointResolution(t *testing.T) {
	for _, tc := range []struct {
		name      string
		endpoints []string
		query     string
		want      string
	}{
		{"exact match", []string{"device", "media"}, "media", "media"},
		// The package is event/, the key GetCapabilities reports is "events". This is the
		// one alias ONVIF actually requires.
		{"package event reaches key events", []string{"device", "events"}, "event", "events"},
		{"absent service", []string{"device", "media"}, "ptz", ""},
		{"no endpoint at all", nil, "device", ""},
		// An exact key must win even though a longer key also contains the name.
		{"exact wins over containing", []string{"device", "analyticsdevice"}, "device", "device"},

		// The case the old substring fallback got wrong. analyticsdevice is the Analytics
		// Device service, a different service from Analytics with a different WSDL; a
		// request from package analytics/ must not be sent to it.
		{"analytics must not reach analyticsdevice", []string{"device", "analyticsdevice"}, "analytics", ""},
		// ...and must still resolve when its own endpoint is there alongside.
		{"analytics resolves when advertised", []string{"analyticsdevice", "analytics"}, "analytics", "analytics"},
		// A vendor-invented key is not a known service, so containment may still reach it.
		{"vendor key reached by containment", []string{"device", "ptzservice"}, "ptz", "ptzservice"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, found := clientWithEndpoints(tc.endpoints...).HasEndpoint(tc.query)
			if tc.want == "" {
				if found {
					t.Fatalf("HasEndpoint(%q) = %q, want not found", tc.query, got)
				}
				return
			}
			if !found {
				t.Fatalf("HasEndpoint(%q) not found, want the %q endpoint", tc.query, tc.want)
			}
			if expected := "http://192.0.2.1/onvif/" + tc.want; got != expected {
				t.Fatalf("HasEndpoint(%q) = %q, want %q", tc.query, got, expected)
			}
		})
	}
}

// TestHasEndpointIsDeterministic pins the tie-break down.
//
// Several vendor keys can contain the service name, and the old implementation returned the
// first one a map range yielded. Go deliberately randomises map iteration order, so that was
// not merely unspecified — the answer varied between runs. Neither candidate here is a known
// ONVIF service, so both survive the exclusion and the tie-break is what decides.
func TestHasEndpointIsDeterministic(t *testing.T) {
	client := clientWithEndpoints("ptzextended", "ptzsvc", "device")

	const want = "http://192.0.2.1/onvif/ptzsvc" // shortest of the two candidates
	for i := 0; i < 200; i++ {
		got, found := client.HasEndpoint("ptz")
		if !found {
			t.Fatal("HasEndpoint(\"ptz\") not found")
		}
		if got != want {
			t.Fatalf("iteration %d: HasEndpoint(\"ptz\") = %q, want %q", i, got, want)
		}
	}
}

// TestGetEndpointReportsErrNoService checks that an absent service is reported as a property
// of the device rather than as a failed exchange, since a caller has to tell a conditional
// service apart from a rejected request.
func TestGetEndpointReportsErrNoService(t *testing.T) {
	client := clientWithEndpoints("device", "media")

	_, err := client.getEndpoint("ptz")
	if err == nil {
		t.Fatal("getEndpoint(\"ptz\") succeeded, want an error")
	}
	if !errors.Is(err, utils.ErrNoService) {
		t.Fatalf("getEndpoint(\"ptz\") = %v, want it to match utils.ErrNoService", err)
	}
	// The service name has to survive into the message: "no endpoint for the service" alone
	// does not say which, and a full dump issues calls to several.
	if want := "ptz"; !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not name the service %q", err, want)
	}
}

// TestGetServicesReturnsACopy pins D1: the accessor used to hand back the client's own
// routing table. CallMethod resolves service names against that map, and
// bin/onvif-cli/dump.go encodes the result straight to stdout, so a caller that adjusted
// what it had printed silently rerouted -- or unrouted -- every later call.
func TestGetServicesReturnsACopy(t *testing.T) {
	client := clientWithEndpoints("device", "media")

	services := client.GetServices()
	if len(services) != 2 {
		t.Fatalf("GetServices returned %d endpoints, want 2", len(services))
	}

	// Everything a caller might plausibly do to a map it believes it owns.
	services["media"] = "http://attacker.example/onvif/media"
	delete(services, "device")
	services["ptz"] = "http://attacker.example/onvif/ptz"

	if got := client.GetEndpoint("media"); got != "http://192.0.2.1/onvif/media" {
		t.Errorf("GetEndpoint(media) = %q after the caller rewrote its copy; the client's "+
			"routing table is reachable from outside", got)
	}
	if _, found := client.HasEndpoint("device"); !found {
		t.Error("device disappeared after the caller deleted it from its copy")
	}
	if _, found := client.HasEndpoint("ptz"); found {
		t.Error("ptz became resolvable after the caller added it to its copy")
	}
}

// AtDeviceHost re-points a URI the device handed out, which ONVIF Core section 9.10.4 shows
// a device filling in from its own point of view: its example answers
// http://160.10.64.10/Subscription?Idx=0, an address that need not resolve from where the
// client sits. AddEndpoint has applied that correction to every advertised XAddr for years.
//
// It cannot apply the same rule, and the port is why. AddEndpoint replaces host and port
// together; a pull-point subscription commonly answers on a port of its own -- the reply
// fixture in event/namespace_test.go has the device on :80 and the subscription on :8000 --
// so taking ours would send every pull to the device service. But keeping the device's port
// unconditionally breaks the very case the rewrite exists for: a camera behind a port map
// advertises its internal port, which is as unusable as the internal host was.
//
// So the host the device named is the discriminator, and these rows are that decision: the
// device naming the host we already reach it at is describing a real port on the machine we
// are talking to, and anything else is describing itself from a vantage point we do not
// share.
func TestAtDeviceHost(t *testing.T) {
	for _, tc := range []struct {
		name  string
		xaddr string
		raw   string
		want  string
	}{
		{
			// The device names itself, on a port of its own: the fixture case, and the one
			// that rules out taking our port.
			"the same host on another port keeps that port",
			"192.168.1.70:80",
			"http://192.168.1.70:8000/onvif/Subscription?Idx=7",
			"http://192.168.1.70:8000/onvif/Subscription?Idx=7",
		},
		{
			"the section 9.10.4 example, advertised from the device's own point of view",
			"192.0.2.1:80",
			"http://160.10.64.10/Subscription?Idx=0",
			"http://192.0.2.1:80/Subscription?Idx=0",
		},
		{
			// A camera behind a port map: reached at :8080, advertising its internal
			// 192.168.1.70:80. Keeping the device's port would produce :80 on the public
			// host, which resolves to nothing -- and this is the case the whole rewrite
			// exists to fix, so it must not be the case it breaks.
			"a foreign host's port is as unusable as its host, so both are ours",
			"203.0.113.5:8080",
			"http://192.168.1.70:80/onvif/Subscription?Idx=0",
			"http://203.0.113.5:8080/onvif/Subscription?Idx=0",
		},
		{
			// Some devices answer with a relative reference. Without a scheme net/url
			// serialises the result as "//host/path", a protocol-relative URI that
			// http.NewRequest cannot post to.
			"a relative reference gains the scheme",
			"192.168.1.70:80",
			"/onvif/Subscription?Idx=7",
			"http://192.168.1.70:80/onvif/Subscription?Idx=7",
		},
		{
			"an IPv6 device naming itself keeps its port and its brackets",
			"[2001:db8::1]:80",
			"http://[2001:db8::1]:8000/onvif/Subscription?Idx=7",
			"http://[2001:db8::1]:8000/onvif/Subscription?Idx=7",
		},
		{
			// Some firmware embeds an account in the address it advertises. It is dropped:
			// we authenticate with a UsernameToken of our own, the manager URI is logged at
			// debug, and net/http would turn req.URL.User into an HTTP Basic Authorization
			// header on every pull.
			"a credential the device embedded is dropped",
			"192.168.1.70:80",
			"http://admin:s3cret@192.168.1.70:8000/onvif/Subscription?Idx=0",
			"http://192.168.1.70:8000/onvif/Subscription?Idx=0",
		},
		{
			"a bare username is dropped too, so nothing of the userinfo survives",
			"192.168.1.70:80",
			"http://admin@160.10.64.10/onvif/Subscription?Idx=0",
			"http://192.168.1.70:80/onvif/Subscription?Idx=0",
		},
		{
			// The device's own answer, returned as it stands: refusing it here would replace
			// a request that might work with an error that cannot.
			"something unparseable comes back untouched",
			"192.168.1.70:80",
			"http://[oops",
			"http://[oops",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(ClientInfo{Xaddr: tc.xaddr}, nil)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if got := client.AtDeviceHost(tc.raw); got != tc.want {
				t.Errorf("AtDeviceHost(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// AddEndpoint has the same exposure and now the same rule: an account a device embedded in
// an advertised XAddr becomes an HTTP Basic Authorization header on every request to that
// service, because net/http fills that in from req.URL.User.
func TestAddEndpointDropsAnEmbeddedCredential(t *testing.T) {
	client, err := NewClient(ClientInfo{Xaddr: "192.168.1.70:80"}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.AddEndpoint("Media", "http://admin:s3cret@192.168.1.70/onvif/media_service")

	got := client.GetEndpoint("media")
	if strings.Contains(got, "s3cret") || strings.Contains(got, "admin") {
		t.Errorf("the endpoint carries the credential the device advertised: %q", got)
	}
	if got != "http://192.168.1.70:80/onvif/media_service" {
		t.Errorf("GetEndpoint(media) = %q, want the service address without the userinfo", got)
	}
}

// GetEndpoint indexed the map directly while getEndpoint, which CallMethod uses, resolved
// through HasEndpoint. Two exported lookups answering the same question differently, and the
// one with the shorter and more obvious name was the wrong one.
//
// The divergence is not hypothetical or rare: ONVIF advertises the event service as "Events",
// AddEndpoint lowercases that to "events", and serviceEndpointKeys exists to map "event" onto
// it. So on an ordinary camera Appliance.GetEndpoint("event") returned "" while HasEvent()
// answered true and CallMethod routed event requests perfectly well. A caller that believed
// the first concluded the camera had no event service.
func TestGetEndpointResolvesTheSameWayCallMethodDoes(t *testing.T) {
	client, err := NewClient(ClientInfo{Xaddr: "10.0.0.1:80"}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// Exactly what a device advertises, capitalised its own way.
	client.AddEndpoint("Events", "http://10.0.0.1/onvif/event_service")

	want, found := client.HasEndpoint("event")
	if !found {
		t.Fatal("HasEndpoint cannot resolve `event` from `Events`, so serviceEndpointKeys has changed")
	}
	if got := client.GetEndpoint("event"); got != want {
		t.Errorf("GetEndpoint(%q) = %q, want %q: it disagrees with the resolution "+
			"CallMethod performs, so a caller reads the service as absent", "event", got, want)
	}

	// A service the device never advertised must still come back empty, or the fix would
	// have traded a false negative for a false positive.
	if got := client.GetEndpoint("ptz"); got != "" {
		t.Errorf("GetEndpoint(%q) = %q, want empty", "ptz", got)
	}
}

// AddEndpoint's rewrite was guarded by `if u, err := url.Parse(Value); err == nil` with no
// else, so a value that would not parse was stored exactly as the device sent it -- skipping
// both things the block exists to do: re-point the host at the address the device is actually
// reachable at, and drop any account it embedded.
//
// url.Parse is permissive but it does fail, and on strings a device can plausibly advertise:
// a stray "%zz" escape, a control character in the path, a trailing bare "%". The credential
// never reached the wire, because SendSoap's http.NewRequestWithContext re-parses the same
// string and fails the same way -- but the call then failed at request time with an opaque
// net/url message instead of here, where the cause is visible, and AGENTS.md's "no ignored
// errors" was breached in the one place that matters.
//
// An endpoint that cannot be parsed cannot be used, so it is not recorded at all: the caller
// gets ErrNoService naming the service, which is the accurate report.
func TestAddEndpointRefusesAnUnparseableXAddr(t *testing.T) {
	for _, bad := range []string{
		"http://user:pass@10.0.0.1/onvif/%zz",
		"http://user:pass@10.0.0.1:80/bad%",
		":://nonsense",
	} {
		t.Run(bad, func(t *testing.T) {
			client, err := NewClient(ClientInfo{Xaddr: "10.0.0.1:80"}, nil)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			client.AddEndpoint("Media", bad)

			if got, found := client.HasEndpoint("media"); found {
				t.Errorf("an unparseable XAddr was recorded as %q; it can never be used, "+
					"and it still carries whatever the device embedded in it", got)
			}
		})
	}

	// A value that parses is still recorded, host rewritten and userinfo dropped.
	client, err := NewClient(ClientInfo{Xaddr: "10.0.0.1:80"}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.AddEndpoint("Media", "http://user:pass@192.168.9.9/onvif/media_service")
	got, found := client.HasEndpoint("media")
	if !found {
		t.Fatal("a well-formed XAddr was dropped along with the malformed ones")
	}
	if want := "http://10.0.0.1:80/onvif/media_service"; got != want {
		t.Errorf("HasEndpoint = %q, want %q", got, want)
	}
}

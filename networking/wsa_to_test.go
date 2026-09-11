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

// CallMethod derived its POST target from the request struct's package name and from
// nothing else, so the five event operations ONVIF addresses to a subscription manager could
// not be sent at all: they need a wsa:To of the URI that
// CreatePullPointSubscriptionResponse handed back, and they must be POSTed there rather than
// to the event service. event/actions.go recorded that as the reason a real subscription
// could not be completed.
//
// Verified against the requests in ONVIF Core sections 9.10.5 and 9.10.7, which carry both
// wsa:Action and wsa:To, and against the one in section 9.10.3, which carries neither --
// which is why the "declares none" half below matters as much as the other.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// The URI shape a device really answers with (ONVIF Core section 9.10.4): a path and a
// query, on a port of its own.
const subscriptionPath = "/onvif/Subscription?Idx=7"

// addressedRequest names its own destination, the way event's manager-addressed requests do.
type addressedRequest struct {
	XMLName string `xml:"tev:PullMessages"`
	To      string `xml:"-"`
}

func (r addressedRequest) WSATo() string { return r.To }

// captureCalls stands up a server recording where each request went and what it carried.
func captureCalls(t *testing.T) (*httptest.Server, *[]string, *[]string) {
	t.Helper()

	paths := make([]string, 0, 1)
	bodies := make([]string, 0, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		paths = append(paths, r.URL.RequestURI())
		bodies = append(bodies, string(body))
		_, _ = w.Write([]byte(`<Envelope/>`))
	}))
	t.Cleanup(srv.Close)
	return srv, &paths, &bodies
}

func TestCallMethodAddressesTheDeclaredTarget(t *testing.T) {
	// Satisfied at compile time, and by a VALUE. CallMethod is handed an interface holding a
	// value -- the generated wrappers pass their request by value -- so a pointer receiver
	// would not be found and every such request would quietly go to the service endpoint
	// instead, which is the failure this mechanism exists to remove.
	var _ WSAAddressee = addressedRequest{}

	srv, paths, bodies := captureCalls(t)
	client, err := NewClient(ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")}, srv.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// CallMethod routes on the last segment of the request type's package path, which for a
	// type declared here is this package. The manager-addressed request must reach its own
	// target without consulting this at all.
	client.AddEndpoint("networking", srv.URL+"/onvif/events")

	resp, err := client.CallMethod(context.Background(),
		addressedRequest{To: srv.URL + subscriptionPath})
	if err != nil {
		t.Fatalf("CallMethod: %v", err)
	}
	_ = resp.Body.Close()

	if len(*paths) != 1 {
		t.Fatalf("the request was not sent: %d requests arrived", len(*paths))
	}
	if got := (*paths)[0]; got != subscriptionPath {
		t.Errorf("the request went to %q, want %q -- a PullMessages POSTed to the event "+
			"service endpoint reaches no subscription", got, subscriptionPath)
	}

	body := (*bodies)[0]
	if n := strings.Count(body, "<wsa:To>"); n != 1 {
		t.Errorf("found %d wsa:To elements, want exactly 1: %s", n, body)
	}
	if !strings.Contains(body, srv.URL+subscriptionPath) {
		t.Errorf("the wsa:To header does not carry the destination: %s", body)
	}
	// Not mustUnderstand, like wsa:Action: a device that ignores WS-Addressing and routes on
	// the POST URL alone must keep working.
	if strings.Contains(body, "mustUnderstand") {
		t.Errorf("wsa:To is marked mustUnderstand: %s", body)
	}
}

func TestCallMethodRoutesOnThePackageWhenNoTargetIsDeclared(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request interface{}
	}{
		// An empty WSATo is what a request built without a manager URI answers, and it has
		// to mean "route normally" or the zero value of every addressed struct is unusable.
		{"an addressed request with no destination set", addressedRequest{}},
		// And a request that implements neither interface must go out exactly as it did
		// before any of this, which is 200 of the 205 operations.
		{"a request declaring nothing at all", plainRequest{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, paths, bodies := captureCalls(t)
			client, err := NewClient(ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")}, srv.Client())
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			client.AddEndpoint("networking", srv.URL+"/onvif/events")

			resp, err := client.CallMethod(context.Background(), tc.request)
			if err != nil {
				t.Fatalf("CallMethod: %v", err)
			}
			_ = resp.Body.Close()

			if got := (*paths)[0]; got != "/onvif/events" {
				t.Errorf("the request went to %q, want the service endpoint /onvif/events", got)
			}
			if strings.Contains((*bodies)[0], "wsa:To") {
				t.Errorf("a wsa:To was added to a request that declares no destination, "+
					"which section 9.10.3 shows carrying none: %s", (*bodies)[0])
			}
		})
	}
}

// The ordering inside methodEndpoint, which is not visible from the happy path: a device may
// grant a subscription and still not advertise the event service in its GetCapabilities, and
// resolving the service first would fail such a camera with ErrNoService on a pull point it
// had just been given.
func TestADeclaredTargetNeedsNoAdvertisedService(t *testing.T) {
	srv, paths, _ := captureCalls(t)
	client, err := NewClient(ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")}, srv.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// No AddEndpoint at all beyond the Device one NewClient installs.

	resp, err := client.CallMethod(context.Background(),
		addressedRequest{To: srv.URL + subscriptionPath})
	if err != nil {
		t.Fatalf("CallMethod: %v -- a declared target must not be resolved through the "+
			"endpoint map", err)
	}
	_ = resp.Body.Close()

	if got := (*paths)[0]; got != subscriptionPath {
		t.Errorf("the request went to %q, want %q", got, subscriptionPath)
	}
}

// A subscription URI is a query string, and nothing in ONVIF stops a device putting an
// ampersand in it. The header is built as an etree element rather than parsed from a string
// fragment precisely so that the escaping is not this package's problem; this is what says
// the choice still holds.
func TestWsaToSurvivesAnAmpersand(t *testing.T) {
	const messy = "/onvif/Subscription?Idx=7&sid=a<b"

	srv, _, bodies := captureCalls(t)
	client, err := NewClient(ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")}, srv.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.CallMethod(context.Background(), addressedRequest{To: srv.URL + messy})
	if err != nil {
		t.Fatalf("CallMethod: %v", err)
	}
	_ = resp.Body.Close()

	// Read it back the way a device would, rather than asserting on the escaped bytes: what
	// matters is that the URI round-trips, not which escaping was used to get it there.
	doc := etree.NewDocument()
	if err := doc.ReadFromString((*bodies)[0]); err != nil {
		t.Fatalf("the envelope does not parse, so the URI was interpolated raw: %v", err)
	}
	to := doc.FindElement("//To")
	if to == nil {
		t.Fatalf("no wsa:To in: %s", (*bodies)[0])
	}
	if got, want := to.Text(), srv.URL+messy; got != want {
		t.Errorf("wsa:To = %q, want %q", got, want)
	}
}

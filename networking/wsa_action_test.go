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

// CallMethod never called gosoap.AddAction, so no request carried a wsa:Action even though
// ONVIF requires WS-Addressing for the event service.
//
// The header is opt-in per request type rather than global, and that restriction is the
// half worth pinning: ONVIF's device, media and PTZ examples carry no addressing header, so
// emitting one for all 205 operations would be a change to every request against devices
// that work today.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// actorRequest declares an action, the way event's request structs do.
type actorRequest struct {
	XMLName string `xml:"tev:PullMessages"`
}

func (actorRequest) WSAAction() string {
	return "http://www.onvif.org/ver10/events/wsdl/PullPointSubscription/PullMessagesRequest"
}

// plainRequest declares none, the way device, media and ptz request structs do.
type plainRequest struct {
	XMLName string `xml:"tds:GetSystemDateAndTime"`
}

func TestCallMethodEmitsWsaActionOnlyWhenDeclared(t *testing.T) {
	// The interface is satisfied at compile time, so a rename cannot silently disable the
	// header for every event operation at once.
	var _ WSAActor = actorRequest{}

	for _, tc := range []struct {
		name    string
		request interface{}
		want    bool
	}{
		{"a request declaring an action", actorRequest{}, true},
		{"a request declaring none", plainRequest{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				buf := make([]byte, r.ContentLength)
				_, _ = r.Body.Read(buf)
				body = string(buf)
				_, _ = w.Write([]byte(`<Envelope/>`))
			}))
			defer srv.Close()

			client, err := NewClient(ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")}, srv.Client())
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			// CallMethod routes on the last segment of the request type's package path,
			// which for these locally declared types is this package.
			client.AddEndpoint("networking", srv.URL)

			resp, err := client.CallMethod(context.Background(), tc.request)
			if err != nil {
				t.Fatalf("CallMethod: %v", err)
			}
			_ = resp.Body.Close()

			got := strings.Contains(body, "Action")
			switch {
			case tc.want && !got:
				t.Errorf("no wsa:Action in the envelope: %s", body)
			case !tc.want && got:
				t.Errorf("a wsa:Action was added to a request that declares none, which "+
					"changes every device, media and ptz request: %s", body)
			}

			if tc.want {
				if n := strings.Count(body, "<wsa:Action>"); n != 1 {
					t.Errorf("found %d wsa:Action elements, want exactly 1: %s", n, body)
				}
				if !strings.Contains(body, actorRequest{}.WSAAction()) {
					t.Errorf("the header does not carry the declared action: %s", body)
				}
				// gosoap deliberately leaves mustUnderstand off, so that a device
				// ignoring WS-Addressing keeps working.
				if strings.Contains(body, "mustUnderstand") {
					t.Errorf("the action is marked mustUnderstand, which would break "+
						"devices that ignore WS-Addressing: %s", body)
				}
			}
		})
	}
}

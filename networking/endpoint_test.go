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

	"github.com/jfsmig/onvif/utils"
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

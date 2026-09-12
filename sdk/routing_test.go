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

package sdk

// CallMethod picks the service endpoint from the last segment of the request struct's
// PkgPath, so the four service *directory* names are entries in a routing table and not
// merely organisation. AGENTS.md says so -- "the directory names device, media, ptz, event
// are load-bearing" -- and credentials/resolver.go writes the rule down defensively in prose,
// but nothing executed it.
//
// The failure it guards against is silent and total. Go permits a package clause that differs
// from its directory -- this tree used to prove it, with Imaging/ holding `package imaging`,
// until that was renamed and scripts/repo-check.sh grew a rule forbidding the mismatch. So
// renaming the directory ptz/ to ptz_service/ while leaving `package ptz` alone compiles,
// vets, and passes every other test in the repository -- while every PTZ request resolves to
// getEndpoint("ptz_service"), yields utils.ErrNoService, is swallowed by FetchPTZ at trace
// level, and prints as `dump ptz` returning nulls with exit 0. Invisible inside the process,
// invisible in CI.
//
// The two checks are complementary and neither replaces the other: repo-check sees that a
// directory and its package agree, and this sees that the name they agree on is one a device
// actually advertises.
//
// It lives in sdk because sdk is the only package that imports all four services and
// networking; the same test in networking would be an import cycle.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/event"
	"github.com/jfsmig/onvif/media"
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/ptz"
)

func TestRequestPackagesNameTheirONVIFService(t *testing.T) {
	for _, tc := range []struct {
		request any
		// advertised is the key a device puts in its GetCapabilities reply, which
		// AddEndpoint lowercases on the way into the map.
		advertised string
	}{
		{device.GetCapabilities{}, "Device"},
		{media.GetProfiles{}, "Media"},
		{ptz.GetNodes{}, "PTZ"},
		// The event service is advertised as `Events`, which is why serviceEndpointKeys
		// maps `event` onto it. Including it here keeps that alias load-bearing too.
		{event.GetEventProperties{}, "Events"},
	} {
		pkg := reflect.TypeOf(tc.request).PkgPath()
		derived := strings.ToLower(pkg[strings.LastIndex(pkg, "/")+1:])

		t.Run(derived, func(t *testing.T) {
			client, err := networking.NewClient(networking.ClientInfo{Xaddr: "10.0.0.1:80"}, nil)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			client.AddEndpoint(tc.advertised, "http://10.0.0.1/onvif/service")

			if _, ok := client.HasEndpoint(derived); !ok {
				t.Fatalf("a request from %s routes to %q, which resolves to no endpoint on a "+
					"device advertising %q: the directory has been renamed away from its ONVIF "+
					"service name, and every call in it now fails with ErrNoService that "+
					"Fetch* swallows into an empty dump",
					pkg, derived, tc.advertised)
			}
		})
	}
}

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

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/ptz"
	"github.com/jfsmig/onvif/utils"
)

// stubAppliance stands up a device answering GetSystemDateAndTime and GetCapabilities,
// advertising exactly the services named. Everything else 500s: these tests are about which
// services are reachable, not about the replies.
func stubAppliance(t *testing.T, services ...string) *networking.Client {
	t.Helper()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		host := strings.TrimPrefix(srv.URL, "http://")

		if !strings.Contains(string(buf), "GetCapabilities") {
			_, _ = w.Write([]byte(soap(`<tds:GetSystemDateAndTimeResponse/>`)))
			return
		}
		var caps strings.Builder
		for _, service := range services {
			caps.WriteString("<tt:" + service + "><tt:XAddr>http://" + host +
				"/onvif/" + strings.ToLower(service) + "_service</tt:XAddr></tt:" + service + ">")
		}
		_, _ = w.Write([]byte(soap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
			caps.String() + `</tds:Capabilities></tds:GetCapabilitiesResponse>`)))
	}))
	t.Cleanup(srv.Close)

	client, err := networking.NewClient(
		networking.ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := WrapClient(context.Background(), client, networking.ClientAuth{}); err != nil {
		t.Fatalf("WrapClient: %v", err)
	}
	return client
}

// TestProfileSGateIgnoresConditionalServices is the regression this design most risks.
//
// Profile S makes PTZ conditional (8.3, "if supported") and event handling conditional
// (7.7), so a camera without a pan-tilt head is a perfectly ordinary Profile S device. If
// NewProfileS gated on every service it mentions, such a camera would get no client at all
// — and the SDK's whole contract is that one unsupported service must not lose the rest.
func TestProfileSGateIgnoresConditionalServices(t *testing.T) {
	client := stubAppliance(t, "Device", "Media")

	profile, ok := NewProfileS(client)
	if !ok {
		t.Fatal("NewProfileS reported false for a device advertising Device and Media, " +
			"which are the only services Profile S requires unconditionally")
	}
	if profile.HasPTZ() {
		t.Error("HasPTZ is true though no PTZ endpoint was advertised")
	}
	if profile.HasEvent() {
		t.Error("HasEvent is true though no event endpoint was advertised")
	}
}

// TestProfileSConditionalOperationReportsErrNoService checks that calling into a service the
// appliance does not have is reported as a property of the device rather than as a failed
// exchange. A caller has to be able to tell "this camera has no PTZ" from "the move was
// rejected", and nothing is sent in the first case.
func TestProfileSConditionalOperationReportsErrNoService(t *testing.T) {
	client := stubAppliance(t, "Device", "Media")

	profile, ok := NewProfileS(client)
	if !ok {
		t.Fatal("NewProfileS reported false")
	}

	_, err := profile.ContinuousMove(context.Background(), ptz.ContinuousMove{})
	if err == nil {
		t.Fatal("ContinuousMove succeeded against a device with no PTZ service")
	}
	if !errors.Is(err, utils.ErrNoService) {
		t.Fatalf("ContinuousMove = %v, want it to match utils.ErrNoService", err)
	}
}

// TestProfileSGateRequiresMandatoryServices is the other half: media is not conditional, so
// an appliance without it is not a Profile S device and must not be presented as one.
//
// Only the media half of the gate is falsifiable here. NewClient seeds a device endpoint
// from the address it was given (networking/client.go:99) before anything is advertised, so
// HasEndpoint("device") is true for every client that exists. That is defensible — without
// device_service there is no ONVIF conversation at all, and WrapClient would already have
// failed with ErrNotOnvif — but it does mean the generated device check can never fire, and
// a test claiming to cover it would be claiming more than it tests.
func TestProfileSGateRequiresMandatoryServices(t *testing.T) {
	for _, tc := range []struct {
		name     string
		services []string
	}{
		{"media missing", []string{"Device"}},
		{"nothing relevant advertised", []string{"Imaging"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := NewProfileS(stubAppliance(t, tc.services...)); ok {
				t.Fatalf("NewProfileS reported true for an appliance advertising only %v", tc.services)
			}
		})
	}
}

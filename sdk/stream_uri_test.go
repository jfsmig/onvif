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

// FetchStreamURI interpolated the username and password into the RTSP URL it returned,
// and picked its profile by map iteration order. Its own doc comment admitted both.
//
// The credential half is the one that matters: AGENTS.md requires that a secret not reach
// a dump, and a URL is exactly the kind of string a caller logs, prints or hands to
// another process. The determinism half is the same defect HasEndpoint was fixed for --
// see networking/endpoint_test.go's TestHasEndpointIsDeterministic, which this mirrors.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jfsmig/onvif/networking"
)

const (
	streamUser = "operator"
	streamPass = "hunter2"
)

// streamStub answers with the given media profiles, each carrying a distinctive stream URI.
func streamStub(t *testing.T, tokens ...string) *httptest.Server {
	t.Helper()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		req := string(buf)
		host := strings.TrimPrefix(srv.URL, "http://")

		var profiles strings.Builder
		for _, tok := range tokens {
			profiles.WriteString(`<tt:Profiles token="` + tok + `"></tt:Profiles>`)
		}

		var out string
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			out = soap(`<tds:GetSystemDateAndTimeResponse/>`)
		case strings.Contains(req, "GetCapabilities"):
			out = soap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
				`<tt:Device><tt:XAddr>http://` + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
				`<tt:Media><tt:XAddr>http://` + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
				`</tds:Capabilities></tds:GetCapabilitiesResponse>`)
		case strings.Contains(req, "GetProfiles"):
			out = soap(`<trt:GetProfilesResponse>` + profiles.String() + `</trt:GetProfilesResponse>`)
		case strings.Contains(req, "GetStreamUri"):
			// The token is not echoed back, so every profile yields the same URI. That is
			// fine: this test is about credentials and stability, not about which URI.
			out = soap(`<trt:GetStreamUriResponse><trt:MediaUri>` +
				`<tt:Uri>rtsp://` + host + `/stream</tt:Uri>` +
				`</trt:MediaUri></trt:GetStreamUriResponse>`)
		default:
			out = soap(`<tds:Empty/>`)
		}
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(out))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func streamProfileS(t *testing.T, srv *httptest.Server) *ProfileS {
	t.Helper()

	dev, err := NewDevice(context.Background(),
		networking.ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")},
		networking.ClientAuth{Username: streamUser, Password: streamPass},
		nil)
	if err != nil {
		t.Fatalf("NewDevice: %v", err)
	}
	profileS, ok := dev.ProfileS()
	if !ok {
		t.Fatal("the stub advertises Device and Media, so Profile S should be available")
	}
	return profileS
}

// TestStreamURICarriesNoCredentials is the one that matters. Credentials are deliberately
// set on the client, so that a regression would have something to leak.
func TestStreamURICarriesNoCredentials(t *testing.T) {
	uri := streamProfileS(t, streamStub(t, "Profile_1")).FetchStreamURI(context.Background())

	if uri == "" {
		t.Fatal("FetchStreamURI returned nothing; the stub is not being reached")
	}
	if strings.Contains(uri, streamPass) {
		t.Errorf("FetchStreamURI = %q carries the password", uri)
	}
	if strings.Contains(uri, streamUser) {
		t.Errorf("FetchStreamURI = %q carries the username", uri)
	}
	// The userinfo separator is the shape the old code produced, so its absence is the
	// clearest single statement that no credential was spliced in.
	if strings.Contains(uri, "@") {
		t.Errorf("FetchStreamURI = %q contains RTSP userinfo", uri)
	}
	if !strings.HasPrefix(uri, "rtsp://") {
		t.Errorf("FetchStreamURI = %q is not the stream URI the device reported", uri)
	}
}

// TestStreamURIIsDeterministic pins the profile choice. Several profiles are advertised in
// an order that is not their sorted order, so a map-range implementation has something to
// get wrong.
//
// 30 iterations, not the 200 networking/endpoint_test.go uses: that test drives a pure
// function, whereas each call here is a full FetchMediaProfiles against the stub. Go
// randomises map iteration per range statement, so with three profiles a map-order
// implementation agreeing 29 times running has probability (1/3)^29 -- the signal is the
// same and the test is not the slowest in the suite.
func TestStreamURIIsDeterministic(t *testing.T) {
	profileS := streamProfileS(t, streamStub(t, "Profile_9", "Profile_2", "Profile_1"))

	first := profileS.FetchStreamURI(context.Background())
	if first == "" {
		t.Fatal("FetchStreamURI returned nothing")
	}
	for i := 0; i < 30; i++ {
		if got := profileS.FetchStreamURI(context.Background()); got != first {
			t.Fatalf("iteration %d returned %q, first returned %q; the profile is chosen "+
				"by map iteration order", i, got, first)
		}
	}
}

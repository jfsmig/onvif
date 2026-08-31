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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jfsmig/onvif/networking"
)

// GetConfiguration and GetConfigurationOptions are keyed by the PTZ *configuration* token,
// not the profile token. The SDK passed the profile token, so a conformant device was asked
// for a configuration that does not exist and PTZ configuration never populated. This test
// drives a stub device and asserts which token actually reaches the wire.

const (
	profileTok = "Profile_1"
	ptzCfgTok  = "PTZCfg_9"
)

func soap(body string) string {
	return `<?xml version="1.0"?><s:Envelope ` +
		`xmlns:s="http://www.w3.org/2003/05/soap-envelope" ` +
		`xmlns:tds="http://www.onvif.org/ver10/device/wsdl" ` +
		`xmlns:trt="http://www.onvif.org/ver10/media/wsdl" ` +
		`xmlns:tptz="http://www.onvif.org/ver20/ptz/wsdl" ` +
		`xmlns:tt="http://www.onvif.org/ver10/schema"><s:Body>` + body + `</s:Body></s:Envelope>`
}

func TestPTZConfigurationIsFetchedByConfigurationToken(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]string{} // operation -> request body

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		req := string(buf)

		host := strings.TrimPrefix(srv.URL, "http://")
		caps := soap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
			`<tt:Device><tt:XAddr>http://` + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
			`<tt:Media><tt:XAddr>http://` + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
			`<tt:PTZ><tt:XAddr>http://` + host + `/onvif/ptz_service</tt:XAddr></tt:PTZ>` +
			`</tds:Capabilities></tds:GetCapabilitiesResponse>`)

		// One profile, whose PTZConfiguration carries a token distinct from the profile's.
		profileXML := `<tt:Profiles token="` + profileTok + `">` +
			`<tt:PTZConfiguration token="` + ptzCfgTok + `"><tt:NodeToken>Node_1</tt:NodeToken></tt:PTZConfiguration>` +
			`</tt:Profiles>`

		var out string
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			out = soap(`<tds:GetSystemDateAndTimeResponse/>`)
		case strings.Contains(req, "GetCapabilities"):
			out = caps
		case strings.Contains(req, "GetProfiles"):
			out = soap(`<trt:GetProfilesResponse>` + profileXML + `</trt:GetProfilesResponse>`)
		case strings.Contains(req, "GetProfile"):
			out = soap(`<trt:GetProfileResponse><trt:Profile token="` + profileTok + `">` +
				`<tt:PTZConfiguration token="` + ptzCfgTok + `"><tt:NodeToken>Node_1</tt:NodeToken></tt:PTZConfiguration>` +
				`</trt:Profile></trt:GetProfileResponse>`)
		case strings.Contains(req, "GetConfigurationOptions"):
			mu.Lock()
			seen["GetConfigurationOptions"] = req
			mu.Unlock()
			out = soap(`<tptz:GetConfigurationOptionsResponse/>`)
		case strings.Contains(req, "GetConfiguration"):
			mu.Lock()
			seen["GetConfiguration"] = req
			mu.Unlock()
			out = soap(`<tptz:GetConfigurationResponse/>`)
		default:
			out = soap(`<tds:Empty/>`)
		}
		w.Header().Set("Content-Type", "application/soap+xml")
		w.Write([]byte(out))
	}))
	defer srv.Close()

	dev, err := NewDevice(context.Background(),
		networking.ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")},
		networking.ClientAuth{}, srv.Client())
	if err != nil {
		t.Fatalf("NewDevice: %v", err)
	}
	dev.FetchProfiles(context.Background())

	mu.Lock()
	defer mu.Unlock()
	for _, op := range []string{"GetConfiguration", "GetConfigurationOptions"} {
		body, ok := seen[op]
		if !ok {
			t.Fatalf("%s was never sent; the SDK is not fetching PTZ configuration at all", op)
		}
		if !strings.Contains(body, ptzCfgTok) {
			t.Fatalf("%s did not carry the PTZ configuration token %q:\n%s", op, ptzCfgTok, body)
		}
		if strings.Contains(body, profileTok) {
			t.Fatalf("%s carried the profile token %q, which is the bug:\n%s", op, profileTok, body)
		}
	}
}

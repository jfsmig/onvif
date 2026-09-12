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

// A device is free to answer GetStreamUri and GetSnapshotUri with the account spliced into
// the URI, and plenty of firmware does: rtsp://admin:secret@192.168.1.70:554/... is the
// ordinary shape. Those two strings are the entire output of `onvif-cli streams` and are
// JSON-encoded wholesale by `dump profile`, `dump media` and `dump all`.
//
// networking.AddEndpoint and AtDeviceHost already drop the userinfo from every other URI a
// device hands over, and networking/endpoint_test.go pins both. These two were the family the
// rule had never reached -- and the one family printed by default.
//
// json:"-" cannot help here, which is why this needs a rule of its own: these are not
// secret-bearing fields, they are ordinary fields whose value happens to contain a secret.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/xsd"
	"github.com/jfsmig/onvif/xsd/onvif"
)

const credentialedPassword = "s3cret"

// credentialedStub answers every URI it hands out with an account embedded, the way firmware
// does, including the XAddrs of its own capabilities.
func credentialedStub(t *testing.T) *ProfileS {
	t.Helper()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = io.ReadFull(r.Body, buf)
		req := string(buf)
		host := strings.TrimPrefix(srv.URL, "http://")
		creds := "admin:" + credentialedPassword + "@"

		var out string
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			out = soap(`<tds:GetSystemDateAndTimeResponse/>`)
		case strings.Contains(req, "GetCapabilities"):
			out = soap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
				`<tt:Device><tt:XAddr>http://` + creds + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
				`<tt:Media><tt:XAddr>http://` + creds + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
				`<tt:Events><tt:XAddr>http://` + creds + host + `/onvif/event_service</tt:XAddr></tt:Events>` +
				`</tds:Capabilities></tds:GetCapabilitiesResponse>`)
		case strings.Contains(req, "GetProfiles"):
			out = soap(`<trt:GetProfilesResponse>` +
				`<tt:Profiles token="Profile_1"></tt:Profiles>` +
				`</trt:GetProfilesResponse>`)
		case strings.Contains(req, "GetStreamUri"):
			out = soap(`<trt:GetStreamUriResponse><trt:MediaUri>` +
				`<tt:Uri>rtsp://` + creds + `192.168.9.9:554/h264Preview_01_main</tt:Uri>` +
				`</trt:MediaUri></trt:GetStreamUriResponse>`)
		case strings.Contains(req, "GetSnapshotUri"):
			out = soap(`<trt:GetSnapshotUriResponse><trt:MediaUri>` +
				`<tt:Uri>http://` + creds + `192.168.9.9:80/snap.jpg</tt:Uri>` +
				`</trt:MediaUri></trt:GetSnapshotUriResponse>`)
		default:
			out = soap(`<tds:Empty/>`)
		}
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(out))
	}))
	t.Cleanup(srv.Close)

	dev, err := NewDevice(context.Background(),
		networking.ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")},
		networking.ClientAuth{Username: "operator", Password: "hunter2"}, nil)
	if err != nil {
		t.Fatalf("NewDevice: %v", err)
	}
	profileS, ok := dev.ProfileS()
	if !ok {
		t.Fatal("the stub advertises Device and Media, so Profile S should be available")
	}
	return profileS
}

func TestStreamAndSnapshotURIsDropTheirUserinfo(t *testing.T) {
	uris := credentialedStub(t).FetchMediaProfileUris(
		context.Background(), ProtocolRTSP, "Profile_1", StreamTypeDefault)

	for _, tc := range []struct {
		name string
		got  string
		// what must survive: the URI is still the operator's answer, only the account goes
		want string
	}{
		{"stream", string(uris.Stream.Uri), "rtsp://192.168.9.9:554/h264Preview_01_main"},
		{"snapshot", string(uris.Snapshot.Uri), "http://192.168.9.9:80/snap.jpg"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got == "" {
				t.Fatal("the stub is not being reached")
			}
			if strings.Contains(tc.got, credentialedPassword) {
				t.Errorf("%q carries the password the device embedded", tc.got)
			}
			if tc.got != tc.want {
				t.Errorf("got %q, want %q: the redaction has to take the account and "+
					"nothing else -- scheme, host, port and path are the answer", tc.got, tc.want)
			}
		})
	}
}

// Same leak, one field over. DeviceDescriptor.Capabilities is the device's own reply handed
// through untouched, and `dump all` / `dump descriptor` encode it whole. Those XAddrs never
// pass through AddEndpoint -- that only ever saw a copy on the way into the routing table --
// so one dump could show the credential stripped from the GetServices() map and preserved in
// the Capabilities block printed a few lines below it.
func TestDumpedCapabilitiesCarryNoUserinfo(t *testing.T) {
	descriptor := credentialedStub(t).FetchDeviceDescriptor(context.Background())

	if descriptor.Capabilities == nil {
		t.Fatal("the stub answers GetCapabilities, so this should not be nil")
	}
	for _, tc := range []struct {
		name string
		got  xsd.AnyURI
	}{
		{"Device", descriptor.Capabilities.Device.XAddr},
		{"Media", descriptor.Capabilities.Media.XAddr},
		{"Events", descriptor.Capabilities.Events.XAddr},
	} {
		if tc.got == "" {
			t.Errorf("%s XAddr is empty; the stub is not being reached", tc.name)
			continue
		}
		if strings.Contains(string(tc.got), credentialedPassword) {
			t.Errorf("%s XAddr = %q carries the password the device embedded", tc.name, tc.got)
		}
		if !strings.HasPrefix(string(tc.got), "http://") || strings.Contains(string(tc.got), "@") {
			t.Errorf("%s XAddr = %q is not the address with only its account removed", tc.name, tc.got)
		}
	}
}

// The three fields above are the three the stub fills. This one covers the other ten, and
// every one added later: it populates every URI in the type, redacts, and looks again.
//
// Reflective on purpose. The whole finding is that one family of URI was missed while two
// others were handled, so an enumeration of today's fields would be the same mistake written
// down. A new capability struct must not be able to reintroduce this quietly.
func TestEveryURIInACapabilitiesIsRedacted(t *testing.T) {
	var capabilities onvif.Capabilities

	const dirty = xsd.AnyURI("http://admin:" + credentialedPassword + "@10.0.0.1/onvif/x")
	filled := forEachURI(reflect.ValueOf(&capabilities), func(reflect.Value) xsd.AnyURI { return dirty })
	if filled < 13 {
		t.Fatalf("only %d URI fields found in onvif.Capabilities; the walk is not reaching them", filled)
	}

	redactURIs(&capabilities)

	var leaked int
	forEachURI(reflect.ValueOf(&capabilities), func(v reflect.Value) xsd.AnyURI {
		got := xsd.AnyURI(v.String())
		if strings.Contains(string(got), "@") {
			leaked++
		}
		return got
	})
	if leaked != 0 {
		t.Errorf("%d of %d URI fields kept their account after redaction", leaked, filled)
	}
}

// forEachURI walks v, replacing every settable xsd.AnyURI with what fn returns for it, and
// reports how many it visited. Used to fill and then to inspect, so the test says nothing
// about how the production walk is written.
func forEachURI(v reflect.Value, fn func(reflect.Value) xsd.AnyURI) int {
	n := 0
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			n += forEachURI(v.Elem(), fn)
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			n += forEachURI(v.Index(i), fn)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			f := v.Field(i)
			if f.Type() == reflect.TypeOf(xsd.AnyURI("")) {
				if f.CanSet() {
					f.SetString(string(fn(f)))
					n++
				}
				continue
			}
			n += forEachURI(f, fn)
		}
	}
	return n
}

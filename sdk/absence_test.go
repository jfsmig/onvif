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

// The Fetch* convention is that a failed sub-call "leaves its field zero". That is only
// truthful where zero cannot be mistaken for an answer the camera gave, and four fields did
// not meet that condition.
//
// sdk/device.go already knows the remedy and applies it to DeviceDescriptor: Capabilities and
// ServiceCapabilities are pointers, so a call that failed is visibly null in a dump rather
// than an empty struct that reads as a reading. These four were the places it had not
// reached, and the bool is the sharpest of them -- every other field of DeviceSecurity is a
// pointer or a slice, so absence was representable for all of them and not for that one.
//
// What it costs to get wrong: a faulted GetClientCertificateMode dumped as false, which an
// operator auditing a fleet reads as "the camera confirmed client certificates are off"; and
// a faulted GetServiceCapabilities on the event service dumped WSPullPointSupport false,
// which a consumer reads as "this camera does not do pull points" and never tries.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jfsmig/onvif/v2/networking"
)

// refusingStub advertises the four services and then faults every operation except the two
// NewDevice needs, which is what a camera that implements only the mandatory core looks like.
func refusingStub(t *testing.T) *ProfileS {
	t.Helper()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = io.ReadFull(r.Body, buf)
		req := string(buf)
		host := strings.TrimPrefix(srv.URL, "http://")

		w.Header().Set("Content-Type", "application/soap+xml")
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			_, _ = w.Write([]byte(soap(`<tds:GetSystemDateAndTimeResponse/>`)))
		case strings.Contains(req, "GetCapabilities"):
			_, _ = w.Write([]byte(soap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
				`<tt:Device><tt:XAddr>http://` + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
				`<tt:Media><tt:XAddr>http://` + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
				`<tt:Events><tt:XAddr>http://` + host + `/onvif/event_service</tt:XAddr></tt:Events>` +
				`<tt:PTZ><tt:XAddr>http://` + host + `/onvif/ptz_service</tt:XAddr></tt:PTZ>` +
				`</tds:Capabilities></tds:GetCapabilitiesResponse>`)))
		default:
			// ONVIF Core section 5.11.2.2 Table 5: an operation the device does not
			// implement is ter:ActionNotSupported.
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(soap(`<env:Fault>` +
				`<env:Code><env:Value>env:Sender</env:Value>` +
				`<env:Subcode><env:Value>ter:ActionNotSupported</env:Value></env:Subcode></env:Code>` +
				`<env:Reason><env:Text xml:lang="en">Operation not supported</env:Text></env:Reason>` +
				`</env:Fault>`)))
		}
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

func TestAFieldNeverAnsweredIsNotReportedAsAValue(t *testing.T) {
	profileS := refusingStub(t)
	ctx := context.Background()

	if got := profileS.FetchDeviceSecurity(ctx).ClientCertificateMode; got != nil {
		t.Errorf("ClientCertificateMode = %v, want nil: GetClientCertificateMode faulted, "+
			"so the camera said nothing, and false reads as a confirmed answer", *got)
	}
	if got := profileS.FetchEvent(ctx).Capabilities; got != nil {
		t.Errorf("Event.Capabilities = %+v, want nil: a struct of false bools reads as "+
			"a camera that supports nothing, including pull points", got)
	}
	if got := profileS.FetchPTZ(ctx).Capabilities; got != nil {
		t.Errorf("Ptz.Capabilities = %+v, want nil", got)
	}
	if got := profileS.FetchMedia(ctx).Capabilities; got != nil {
		t.Errorf("Media.Capabilities = %+v, want nil", got)
	}
}

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

package device

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
)

// Of the 169 colon-form tags in this file, 161 sit on request types and are marshal-only:
// they already serialise to correct wire XML, so rewriting them buys nothing and would
// change every device request. The 8 that matter are on StorageConfiguration and its
// children, which arrive INSIDE GetStorageConfigurationsResponse and so must unmarshal.

const storageResponse = `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
            xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
 <s:Body>
  <tds:GetStorageConfigurationsResponse>
   <tds:StorageConfigurations token="disk0">
     <tds:Data>
       <tds:LocalPath>/mnt/sd0</tds:LocalPath>
       <tds:StorageUri>nfs://192.168.1.9/export</tds:StorageUri>
       <tds:User>
         <tds:UserName>storage-account</tds:UserName>
         <tds:Password>STORAGE-SECRET</tds:Password>
       </tds:User>
     </tds:Data>
   </tds:StorageConfigurations>
  </tds:GetStorageConfigurationsResponse>
 </s:Body>
</s:Envelope>`

func TestStorageConfigurationUnmarshals(t *testing.T) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetStorageConfigurationsResponse GetStorageConfigurationsResponse
		}
	}
	var e Envelope
	if err := xml.Unmarshal([]byte(storageResponse), &e); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	got := e.Body.GetStorageConfigurationsResponse.StorageConfigurations
	if len(got) != 1 {
		t.Fatalf("got %d storage configurations, want 1", len(got))
	}
	if p := string(got[0].Data.LocalPath); p != "/mnt/sd0" {
		t.Fatalf("Data.LocalPath = %q, want %q", p, "/mnt/sd0")
	}
	if u := string(got[0].Data.User.UserName); u != "storage-account" {
		t.Fatalf("Data.User.UserName = %q, want %q", u, "storage-account")
	}
}

// sdk/device.go puts StorageConfigurations into the dump, so this password reaches
// json.Encoder just like the ones in xsd/onvif.
func TestStorageCredentialPasswordIsNotSerialisedToJSON(t *testing.T) {
	const secret = "STORAGE-SECRET"
	b, err := json.Marshal(UserCredential{UserName: "storage-account", Password: secret})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(b), secret) {
		t.Fatalf("storage credential leaked into JSON: %s", b)
	}
	if !strings.Contains(string(b), "storage-account") {
		t.Fatalf("redaction went too far, username lost: %s", b)
	}
	// The device still has to receive it on the request path.
	x, err := xml.Marshal(UserCredential{Password: secret})
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	if !strings.Contains(string(x), secret) {
		t.Fatalf("xml lost the password: %s", x)
	}
}

// The request-only tags are deliberately left in colon form; this pins the wire format so
// a future blanket rewrite cannot change it silently.
func TestDeviceRequestWireFormatUnchanged(t *testing.T) {
	b, err := xml.Marshal(GetCapabilities{Category: "All"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(b); !strings.Contains(got, "<tds:GetCapabilities>") ||
		!strings.Contains(got, "<tds:Category>All</tds:Category>") {
		t.Fatalf("device request wire format changed: %s", got)
	}
}

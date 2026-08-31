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

package onvif

import (
	"encoding/xml"
	"strings"
	"testing"
)

// These are the ONVIF common-schema types, which arrive nested inside almost every
// response. A real camera prefixes them tt:, bound to http://www.onvif.org/ver10/schema —
// verified against 192.168.1.70, which answers GetCapabilities with
// <tds:Capabilities><tt:Analytics><tt:XAddr>...
//
// The tags read `xml:"onvif:XAddr"`. Go's tag separator is a space, not a colon, so the
// whole string became the element name and <tt:XAddr> never matched. Fields with no tag
// bound by local name, which is why responses came back half-populated.

const ttNS = "http://www.onvif.org/ver10/schema"

// wrap builds a fragment in the namespace a device really uses.
func wrap(elem, inner string) string {
	return `<` + elem + ` xmlns:tt="` + ttNS + `">` + inner + `</` + elem + `>`
}

func TestUserUnmarshalsFromDeviceNamespace(t *testing.T) {
	doc := wrap("User", `<tt:Username>admin</tt:Username><tt:Password>PW</tt:Password><tt:UserLevel>Administrator</tt:UserLevel>`)
	var u User
	if err := xml.Unmarshal([]byte(doc), &u); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if u.Username != "admin" {
		t.Fatalf("Username = %q, want %q", u.Username, "admin")
	}
	if string(u.UserLevel) != "Administrator" {
		t.Fatalf("UserLevel = %q, want %q", u.UserLevel, "Administrator")
	}
}

// A nested, multi-level case: this is the shape that made half the dump come back empty.
func TestNestedNetworkInterfaceUnmarshals(t *testing.T) {
	doc := wrap("NetworkInterface", `
		<tt:Enabled>true</tt:Enabled>
		<tt:Info><tt:Name>eth0</tt:Name><tt:HwAddress>00:11:22:33:44:55</tt:HwAddress><tt:MTU>1500</tt:MTU></tt:Info>`)
	var n NetworkInterface
	if err := xml.Unmarshal([]byte(doc), &n); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := string(n.Info.Name); got != "eth0" {
		t.Fatalf("Info.Name = %q, want %q", got, "eth0")
	}
	if got := string(n.Info.HwAddress); got != "00:11:22:33:44:55" {
		t.Fatalf("Info.HwAddress = %q, want %q", got, "00:11:22:33:44:55")
	}
}

// The one namespaced ATTRIBUTE in the file — attributes take the same space form, and this
// is the only place it is exercised.
func TestNamespacedAttributeUnmarshals(t *testing.T) {
	doc := `<BinaryData xmlns:xmime="http://www.w3.org/2005/05/xmlmime" xmime:contentType="image/jpeg"><Data>abc</Data></BinaryData>`
	var b BinaryData
	if err := xml.Unmarshal([]byte(doc), &b); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := string(b.X); got != "image/jpeg" {
		t.Fatalf("contentType attr = %q, want %q", got, "image/jpeg")
	}
}

// Requests carry these types too (SetNetworkInterfaces and friends), so the namespace must
// still be present on the way out — same URI as before, just serialised differently.
func TestMarshalKeepsTheSchemaNamespace(t *testing.T) {
	b, err := xml.Marshal(User{Username: "admin", UserLevel: UserLevel("Administrator")})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(b), ttNS) {
		t.Fatalf("marshalled User lost the ONVIF schema namespace: %s", b)
	}
}

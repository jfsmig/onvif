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

// --- attributes ---
//
// onvif.xsd declares no attributeFormDefault, so it takes the XSD default of "unqualified":
// a locally declared attribute belongs to NO namespace. The one exception in the whole
// schema is xmime:contentType, which is pulled in by ref= and therefore IS qualified.
//
// SimpleItem carries ONVIF's generic name/value pairs — every analytics module, rule and
// metadata configuration passes its parameters through it — so getting this wrong breaks
// configuration in both directions.

// A camera sends the attributes unqualified, exactly as the schema specifies.
const analyticsModule = `<Config xmlns:tt="http://www.onvif.org/ver10/schema" Name="MotionDetector" Type="tt:Motion">
  <tt:Parameters>
    <tt:SimpleItem Name="Sensitivity" Value="70"/>
  </tt:Parameters>
</Config>`

func TestSimpleItemAttributesRoundTrip(t *testing.T) {
	var c Config
	if err := xml.Unmarshal([]byte(analyticsModule), &c); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if c.Name != "MotionDetector" {
		t.Fatalf("Config.Name = %q, want MotionDetector", c.Name)
	}
	// Indexed since ItemList went plural: ONVIF Core section 9.4.1 gives each group an
	// arbitrary number of items, and a single value kept only the last one it met.
	if got := len(c.Parameters.SimpleItem); got != 1 {
		t.Fatalf("Parameters.SimpleItem has %d items, want 1", got)
	}
	si := c.Parameters.SimpleItem[0]
	if si.Name != "Sensitivity" {
		t.Fatalf("SimpleItem.Name = %q, want Sensitivity — analytics parameters do not bind", si.Name)
	}
	if got := string(si.Value); got != "70" {
		t.Fatalf("SimpleItem.Value = %q, want 70", got)
	}
}

// And on the way out the attributes must be unqualified too, or a camera looking for a
// plain Name= will not find it.
func TestSimpleItemMarshalsUnqualifiedAttributes(t *testing.T) {
	b, err := xml.Marshal(SimpleItem{Name: "Sensitivity", Value: "70"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, ":Name=") || strings.Contains(got, ":Value=") {
		t.Fatalf("SimpleItem attributes are namespace-qualified, but the schema declares them local: %s", got)
	}
	for _, want := range []string{`Name="Sensitivity"`, `Value="70"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in: %s", want, got)
		}
	}
}

// AttachmentData carries the same `ref="xmime:contentType"` as BinaryData, so it must be
// qualified the same way. It was the only tag in the file tagged the other way round.
//
// Note the asymmetry that hid this: Go's unmarshal matches on local name alone when the tag
// omits a namespace, so reading already worked. Only the marshal side was wrong, emitting a
// bare contentType= where the schema calls for the xmime-qualified attribute.
func TestAttachmentDataContentTypeIsQualified(t *testing.T) {
	b, err := xml.Marshal(AttachmentData{ContentType: "image/jpeg"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(b), "http://www.w3.org/2005/05/xmlmime") {
		t.Fatalf("AttachmentData.ContentType is not xmime-qualified on the wire: %s", b)
	}
	// Reading is lenient either way, but assert it keeps working.
	const doc = `<AttachmentData xmlns:xmime="http://www.w3.org/2005/05/xmlmime" xmime:contentType="image/jpeg"/>`
	var a AttachmentData
	if err := xml.Unmarshal([]byte(doc), &a); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := string(a.ContentType); got != "image/jpeg" {
		t.Fatalf("AttachmentData.ContentType = %q, want image/jpeg", got)
	}
}

// BinaryData was already right; make sure fixing its twin does not "normalise" it away.
func TestBinaryDataContentTypeStaysQualified(t *testing.T) {
	const doc = `<BinaryData xmlns:xmime="http://www.w3.org/2005/05/xmlmime" xmime:contentType="image/png"/>`
	var b BinaryData
	if err := xml.Unmarshal([]byte(doc), &b); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := string(b.X); got != "image/png" {
		t.Fatalf("BinaryData.X = %q, want image/png", got)
	}
}

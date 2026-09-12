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

package media

import (
	"encoding/xml"
	"strings"
	"testing"
)

// The XMLName tag names the operation element that goes on the wire. It read
// trt:GetDeviceInformation — a copy-paste slip, with the sibling fields correctly trt:
// prefixed — so calling SetMetadataConfiguration actually asked the device for its
// information. Verified against docs/wsdl/media.wsdl.
func TestSetMetadataConfigurationNamesItsOwnOperation(t *testing.T) {
	b, err := xml.Marshal(SetMetadataConfiguration{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, "GetDeviceInformation") {
		t.Fatalf("SetMetadataConfiguration still serialises as GetDeviceInformation: %s", got)
	}
	if !strings.HasPrefix(got, "<trt:SetMetadataConfiguration>") {
		t.Fatalf("operation element is not trt:SetMetadataConfiguration: %s", got)
	}
}

// docs/wsdl/media.wsdl declares every Get*ConfigurationOptions request as
//
//	<xs:sequence>
//	  <xs:element name="ConfigurationToken" type="tt:ReferenceToken" minOccurs="0"/>
//	  <xs:element name="ProfileToken"       type="tt:ReferenceToken" minOccurs="0"/>
//	</xs:sequence>
//
// -- media.wsdl:1383 for GetVideoEncoderConfigurationOptions, and the same shape for the six
// siblings. All seven Go structs listed ProfileToken first, which is two faults on one
// request in a schema whose elementFormDefault is "qualified" (media.wsdl:13), where the
// order of an xs:sequence is normative:
//
//   - the children went out transposed, so a device walking the sequence finds ProfileToken
//     where it expects ConfigurationToken and answers ter:WellFormed or ter:TagMismatch;
//   - both are minOccurs="0" and neither was omitempty, so the one the caller left unset went
//     out as an empty element, which a tolerant device reads as a request for the
//     configuration whose token is "" and answers ter:InvalidArgVal.
//
// sdk/media.go calls two of these with ConfigurationToken alone -- that is the path
// `onvif-cli dump media` reaches -- and sdk.Fetch* swallows the fault at trace level, so the
// whole thing surfaced as a camera reporting no encoder options at all.
func TestConfigurationOptionsFollowTheWSDLSequence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request any
	}{
		{"VideoSource", GetVideoSourceConfigurationOptions{ConfigurationToken: "C1"}},
		{"VideoEncoder", GetVideoEncoderConfigurationOptions{ConfigurationToken: "C1"}},
		{"AudioSource", GetAudioSourceConfigurationOptions{ConfigurationToken: "C1"}},
		{"AudioEncoder", GetAudioEncoderConfigurationOptions{ConfigurationToken: "C1"}},
		{"Metadata", GetMetadataConfigurationOptions{ConfigurationToken: "C1"}},
		{"AudioOutput", GetAudioOutputConfigurationOptions{ConfigurationToken: "C1"}},
		{"AudioDecoder", GetAudioDecoderConfigurationOptions{ConfigurationToken: "C1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := xml.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			got := string(b)
			if strings.Contains(got, "ProfileToken") {
				t.Errorf("an unset optional ProfileToken is still on the wire: %s", got)
			}
			if !strings.Contains(got, "<trt:ConfigurationToken>C1</trt:ConfigurationToken>") {
				t.Errorf("missing trt:ConfigurationToken: %s", got)
			}
		})
	}

	// And when a caller sets both, ConfigurationToken must come first.
	b, err := xml.Marshal(GetVideoEncoderConfigurationOptions{
		ConfigurationToken: "C1", ProfileToken: "P1",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	cfg, prof := strings.Index(got, "ConfigurationToken"), strings.Index(got, "ProfileToken")
	if cfg < 0 || prof < 0 || cfg > prof {
		t.Fatalf("children are out of xs:sequence order, want ConfigurationToken then ProfileToken: %s", got)
	}
}

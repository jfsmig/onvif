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

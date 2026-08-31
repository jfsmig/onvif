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

package ptz

import (
	"encoding/xml"
	"strings"
	"testing"
)

// PresetTourToken and Operation are LOCAL elements of a tptz operation in a schema with
// elementFormDefault="qualified" (docs/wsdl/ptz.wsdl), so they belong to tptz. They were
// tagged onvif:, which is the namespace of their Go *type*, not of the element — the same
// type-vs-element confusion found in device/types.go and event/operation.go.
func TestOperatePresetTourChildrenUseTheServiceNamespace(t *testing.T) {
	b, err := xml.Marshal(OperatePresetTour{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, "onvif:") {
		t.Fatalf("OperatePresetTour still emits onvif:-prefixed children: %s", got)
	}
	for _, want := range []string{"<tptz:PresetTourToken>", "<tptz:Operation>"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in: %s", want, got)
		}
	}
}

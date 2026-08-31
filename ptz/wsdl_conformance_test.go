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

// docs/wsdl/ptz.wsdl declares GetConfiguration's child as PTZConfigurationToken and
// GetConfigurationOptions' as ConfigurationToken. Both structs sent ProfileToken, so the
// device received no token it recognised — and the field name also mislabelled what these
// operations take, which is a configuration token, not a profile token.
func TestGetConfigurationSendsTheConfigurationToken(t *testing.T) {
	b, err := xml.Marshal(GetConfiguration{PTZConfigurationToken: "PTZCfg_9"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, "ProfileToken") {
		t.Fatalf("GetConfiguration still sends ProfileToken: %s", got)
	}
	if !strings.Contains(got, "<tptz:PTZConfigurationToken>PTZCfg_9</tptz:PTZConfigurationToken>") {
		t.Fatalf("missing tptz:PTZConfigurationToken: %s", got)
	}
}

func TestGetConfigurationOptionsSendsTheConfigurationToken(t *testing.T) {
	b, err := xml.Marshal(GetConfigurationOptions{ConfigurationToken: "PTZCfg_9"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, "ProfileToken") {
		t.Fatalf("GetConfigurationOptions still sends ProfileToken: %s", got)
	}
	if !strings.Contains(got, "<tptz:ConfigurationToken>PTZCfg_9</tptz:ConfigurationToken>") {
		t.Fatalf("missing tptz:ConfigurationToken: %s", got)
	}
}

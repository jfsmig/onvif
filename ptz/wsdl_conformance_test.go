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

	"github.com/jfsmig/onvif/xsd"
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

// docs/wsdl/ptz.wsdl:540 gives Stop's PanTilt and Zoom minOccurs="0", and documents the
// absence -- not the value false -- as the instruction to stop:
//
//	"Set true when we want to stop ongoing pan and tilt movements. If PanTilt arguments are
//	 not present, this command stops these movements."
//
// Both were plain xsd.Boolean value fields, so the zero ptz.Stop -- which is what
// sdk.ProfileS.Stop is handed, since the generated wrapper takes the request by value --
// went out as <tptz:PanTilt>false</tptz:PanTilt><tptz:Zoom>false</tptz:Zoom>: a well-formed
// request to stop neither axis. The device answers StopResponse and the camera keeps moving.
//
// omitempty cannot express this. "false" and "absent" are different instructions and a
// two-state bool has nowhere to put the third state, which is why these are pointers.
func TestStopWithNoAxisNamedStopsEverything(t *testing.T) {
	b, err := xml.Marshal(Stop{ProfileToken: "Profile_1"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	for _, dead := range []string{"<tptz:PanTilt>", "<tptz:Zoom>"} {
		if strings.Contains(got, dead) {
			t.Fatalf("the zero Stop still names an axis, so it stops nothing: %s", got)
		}
	}

	// Naming one axis must still name it, or the pointer would have traded one silent
	// failure for another.
	yes := xsd.Boolean(true)
	b, err = xml.Marshal(Stop{ProfileToken: "Profile_1", PanTilt: &yes})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(b); !strings.Contains(got, "<tptz:PanTilt>true</tptz:PanTilt>") {
		t.Fatalf("an explicit PanTilt stop did not reach the wire: %s", got)
	}
}

// docs/wsdl/ptz.wsdl gives Speed minOccurs="0" on all five move operations, and says why:
// "The speed parameter can only be specified when Speed Spaces are available for the PTZ
// Node." A value-kind onvif.PTZSpeed is always emitted, so every one of these carried a
// <tptz:Speed> holding a zero PanTilt and Zoom -- a parameter the WSDL forbids on a node
// with no speed space, and a request to move at velocity zero on a node that has one.
//
// ContinuousMove.Timeout is the same shape for a different reason: xsd.Duration is a string
// kind whose zero value is "", so an unset Timeout went out as <tptz:Timeout></tptz:Timeout>
// and the empty string is not in the lexical space of xs:duration, which requires P and at
// least one component. There omitempty is enough, because absent is the only other state.
func TestOptionalMoveParametersAreOmittedWhenUnset(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request any
		absent  string
	}{
		{"GotoPreset", GotoPreset{ProfileToken: "P1", PresetToken: "Preset_3"}, "Speed"},
		{"GotoHomePosition", GotoHomePosition{ProfileToken: "P1"}, "Speed"},
		{"RelativeMove", RelativeMove{ProfileToken: "P1"}, "Speed"},
		{"AbsoluteMove", AbsoluteMove{ProfileToken: "P1"}, "Speed"},
		{"GeoMove", GeoMove{ProfileToken: "P1"}, "Speed"},
		{"ContinuousMove", ContinuousMove{ProfileToken: "P1"}, "Timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := xml.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			got := string(b)
			if strings.Contains(got, tc.absent) {
				t.Errorf("an unset optional %s is still on the wire: %s", tc.absent, got)
			}
			if !strings.Contains(got, "<tptz:ProfileToken>P1</tptz:ProfileToken>") {
				t.Errorf("the mandatory ProfileToken did not survive: %s", got)
			}
		})
	}
}

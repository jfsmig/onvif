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

// SystemDateTime.UTCDateTime and .LocalDateTime were typed xsd.DateTime, a string alias,
// while docs/wsdl/devicemgmt.wsdl:447 declares the reply as type="tt:SystemDateTime" whose
// date members are tt:DateTime -- the structured element with Date and Time children.
// encoding/xml collects only character data when the target is of string kind, so both
// fields bound to the whitespace between <tt:Date> and <tt:Time>: they were empty for every
// camera ever queried, and `onvif-cli dump device` printed a blank clock.
//
// docs/wsdl/devicemgmt.wsdl:2688 is the clause that makes this worth a test rather than a
// shrug: "A device shall provide the UTCDateTime information."
//
// Nothing caught it because the only stubs answering the operation, in sdk/profile_s_test.go
// and sdk/ptz_token_test.go, reply with an empty <tds:GetSystemDateAndTimeResponse/>.

import (
	"encoding/xml"
	"testing"

	"github.com/jfsmig/onvif/xsd"
)

// TestSystemDateTimeUnmarshalsTheStructuredUTCDateTime feeds the shape a real camera
// sends: every child qualified in the schema namespace, because onvif.xsd is
// elementFormDefault="qualified".
func TestSystemDateTimeUnmarshalsTheStructuredUTCDateTime(t *testing.T) {
	body := wrap("tds:SystemDateAndTime", `
		<tt:DateTimeType>NTP</tt:DateTimeType>
		<tt:DaylightSavings>true</tt:DaylightSavings>
		<tt:TimeZone><tt:TZ>CET-1CEST,M3.5.0,M10.5.0/3</tt:TZ></tt:TimeZone>
		<tt:UTCDateTime>
			<tt:Time><tt:Hour>14</tt:Hour><tt:Minute>5</tt:Minute><tt:Second>6</tt:Second></tt:Time>
			<tt:Date><tt:Year>2026</tt:Year><tt:Month>8</tt:Month><tt:Day>31</tt:Day></tt:Date>
		</tt:UTCDateTime>
		<tt:LocalDateTime>
			<tt:Time><tt:Hour>16</tt:Hour><tt:Minute>5</tt:Minute><tt:Second>6</tt:Second></tt:Time>
			<tt:Date><tt:Year>2026</tt:Year><tt:Month>8</tt:Month><tt:Day>31</tt:Day></tt:Date>
		</tt:LocalDateTime>`)

	var got SystemDateTime
	if err := xml.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	for _, tc := range []struct {
		field string
		got   xsd.Int
		want  xsd.Int
	}{
		{"UTCDateTime.Date.Year", got.UTCDateTime.Date.Year, 2026},
		{"UTCDateTime.Date.Month", got.UTCDateTime.Date.Month, 8},
		{"UTCDateTime.Date.Day", got.UTCDateTime.Date.Day, 31},
		{"UTCDateTime.Time.Hour", got.UTCDateTime.Time.Hour, 14},
		{"UTCDateTime.Time.Minute", got.UTCDateTime.Time.Minute, 5},
		{"UTCDateTime.Time.Second", got.UTCDateTime.Time.Second, 6},
		// LocalDateTime too: it shared the type, so it shared the defect.
		{"LocalDateTime.Time.Hour", got.LocalDateTime.Time.Hour, 16},
		{"LocalDateTime.Date.Year", got.LocalDateTime.Date.Year, 2026},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.field, tc.got, tc.want)
		}
	}

	// The siblings, so that adding the tags did not cost the fields that already worked.
	if got.TimeZone.TZ != "CET-1CEST,M3.5.0,M10.5.0/3" {
		t.Errorf("TimeZone.TZ = %q, want the POSIX TZ string", got.TimeZone.TZ)
	}
	if got.DaylightSavings != true {
		t.Errorf("DaylightSavings = %v, want true", got.DaylightSavings)
	}
	if got.DateTimeType != "NTP" {
		t.Errorf("DateTimeType = %q, want NTP", got.DateTimeType)
	}
}

// TestSystemDateTimeToleratesAnAbsentUTCDateTime pins the other half. The element is
// optional in the schema, and the clock-offset bootstrap in sdk keys on a zero Year to
// mean "the device said nothing", so a missing element must leave the zero value rather
// than fail the parse.
func TestSystemDateTimeToleratesAnAbsentUTCDateTime(t *testing.T) {
	body := wrap("tds:SystemDateAndTime", `<tt:DateTimeType>Manual</tt:DateTimeType>`)

	var got SystemDateTime
	if err := xml.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.UTCDateTime.Date.Year != 0 {
		t.Errorf("UTCDateTime.Date.Year = %d, want 0 for an absent element",
			got.UTCDateTime.Date.Year)
	}
}

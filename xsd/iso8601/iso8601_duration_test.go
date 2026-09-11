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

package iso8601

// ISO8601Duration emitted its time section only when hours AND minutes AND seconds were all
// set, and then fell into the "P alone" fallback -- so a duration carrying only seconds came
// back as "PT0S". Ten seconds rendered as zero.
//
// It matters because xsd.Duration has no other constructor, and because a PullMessages
// Timeout (ONVIF Core section 9.1.2, docs/wsdl/event.wsdl:129) is exactly a number of
// seconds. Nothing in the tree called it with that shape, so the defect was reachable only
// by the next caller.
//
// Verified against the xs:duration lexical space of XML Schema Part 2 section 3.2.6,
// PnYnMnDTnHnMnS, where each component is optional and the T separates the date components
// from the time ones.

import "testing"

func TestISO8601Duration(t *testing.T) {
	for _, tc := range []struct {
		name                                         string
		years, months, days, hours, minutes, seconds string
		want                                         string
	}{
		{"seconds alone, a PullMessages Timeout", "", "", "", "", "", "10", "PT10S"},
		{"hours alone", "", "", "", "1", "", "", "PT1H"},
		{"minutes alone", "", "", "", "", "5", "", "PT5M"},
		{"minutes and seconds, no hours", "", "", "", "", "1", "30", "PT1M30S"},
		{"date components alone take no T", "1", "2", "3", "", "", "", "P1Y2M3D"},
		{"every component", "1", "2", "3", "4", "5", "6", "P1Y2M3DT4H5M6S"},
		{"days and seconds", "", "", "2", "", "", "30", "P2DT30S"},
		// The zero case keeps its old answer: "P" alone is not a duration, and PT0S is the
		// shortest thing that is.
		{"nothing at all is still a duration", "", "", "", "", "", "", "PT0S"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewDuration(tc.years, tc.months, tc.days, tc.hours, tc.minutes, tc.seconds)
			if err != nil {
				t.Fatalf("NewDuration: %v", err)
			}
			if got := d.ISO8601Duration(); got != tc.want {
				t.Errorf("ISO8601Duration() = %q, want %q", got, tc.want)
			}
		})
	}
}

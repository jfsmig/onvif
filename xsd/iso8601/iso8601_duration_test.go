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

import (
	"strings"
	"testing"
)

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

// TestNewDurationRejectsPartialMatches pins the other half of this constructor: it validates,
// and it was validating almost nothing.
//
// The two patterns read `^$|[0-9]+` and `^$|[0-9]+(\.[0-9]+)?`. Alternation binds at the top
// level, so the anchors belong to the empty-string branch alone and the digit branch carries
// none -- and regexp.MatchString searches rather than matches. Any subject holding a digit run
// anywhere therefore passed, and ISO8601Duration concatenates its components verbatim, so
// "12abc" years came back as "P12abcY" and "-5" as "P-5Y".
//
// Verified against the xs:duration lexical space, XML Schema Part 2 section 3.2.6: each
// component is an unsigned integer and only the seconds component may carry a decimal
// fraction. A schema pattern constrains the whole literal, which is what the anchors say --
// the same slip, and the same reasoning, as the one recorded above NewLanguage in
// xsd/built_in.go.
//
// Reachable, despite having no caller in this repository: xsd.Duration.NewDuration is
// exported, documented, and delegates straight to here.
func TestNewDurationRejectsPartialMatches(t *testing.T) {
	// Each of these is a whole component value, not a fragment: every one was accepted.
	for _, bad := range []string{"12abc", "abc12", "-5", " 5", "1.5", "+5"} {
		t.Run("years/"+bad, func(t *testing.T) {
			if _, err := NewDuration(bad, "", "", "", "", ""); err == nil {
				t.Errorf("NewDuration accepted %q as a number of years", bad)
			}
		})
	}

	// The seconds component is the only one allowed a fraction, and one fraction only.
	for _, bad := range []string{"1.2.3", "1.", ".5", "x1", "1x"} {
		t.Run("seconds/"+bad, func(t *testing.T) {
			if _, err := NewDuration("", "", "", "", "", bad); err == nil {
				t.Errorf("NewDuration accepted %q as a number of seconds", bad)
			}
		})
	}

	// Anchoring must not cost the accept path: an empty component means absent, and the
	// seconds fraction stays legal.
	if _, err := NewDuration("", "", "", "", "", "1.5"); err != nil {
		t.Errorf("NewDuration rejected a fractional number of seconds: %v", err)
	}
	if _, err := NewDuration("1", "2", "3", "4", "5", "6"); err != nil {
		t.Errorf("NewDuration rejected a well-formed duration: %v", err)
	}
}

// TestNewDurationNamesTheComponentItRejected pins four copy-pasted diagnostics: days, hours
// and minutes all reported "months value = ", and seconds reported "years value = ". The
// entire output of a validator is its diagnostic, so naming the wrong component defeats it --
// and it would misdirect the first person to meet the anchoring fix above.
func TestNewDurationNamesTheComponentItRejected(t *testing.T) {
	for _, tc := range []struct {
		name string
		args [6]string
	}{
		{"years", [6]string{"x", "", "", "", "", ""}},
		{"months", [6]string{"", "x", "", "", "", ""}},
		{"days", [6]string{"", "", "x", "", "", ""}},
		{"hours", [6]string{"", "", "", "x", "", ""}},
		{"minutes", [6]string{"", "", "", "", "x", ""}},
		{"seconds", [6]string{"", "", "", "", "", "x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := tc.args
			_, err := NewDuration(a[0], a[1], a[2], a[3], a[4], a[5])
			if err == nil {
				t.Fatalf("NewDuration accepted %q as a number of %s", "x", tc.name)
			}
			if !strings.Contains(err.Error(), tc.name) {
				t.Errorf("rejecting the %s component reported %q", tc.name, err)
			}
		})
	}
}

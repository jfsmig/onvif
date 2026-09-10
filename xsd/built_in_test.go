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

package xsd

import (
	"strings"
	"testing"
	"time"
)

// reference is the instant every constructor below is given. Deliberately not "now":
// the broken layouts these tests were written against ignored their argument entirely,
// so the assertions have to name a date that could only come from the argument.
var reference = time.Date(2026, time.August, 31, 14, 5, 6, 0, time.UTC)

// TestNewDurationReportsAMalformedComponent pins the fix for the one breach AGENTS.md
// named: this constructor called log.Fatalln on a malformed component, which terminates
// the calling process from inside a library. The test passing at all is the assertion --
// under the old code the test binary died here and reported no failure, just a non-zero
// exit.
func TestNewDurationReportsAMalformedComponent(t *testing.T) {
	if _, err := Duration("").NewDuration("x", "0", "0", "0", "0", "0"); err == nil {
		t.Fatal("NewDuration accepted a non-numeric year; expected an error")
	}

	// The success path must still produce an ISO 8601 duration, so that returning an error
	// did not come at the cost of the value.
	d, err := Duration("").NewDuration("1", "2", "3", "4", "5", "6")
	if err != nil {
		t.Fatalf("NewDuration rejected a well-formed duration: %v", err)
	}
	if !strings.HasPrefix(string(d), "P") {
		t.Errorf("NewDuration = %q, want an ISO 8601 duration starting with P", d)
	}
}

// TestConstructorsReflectTheirArgument pins the two layout strings that were not Go
// layouts at all. Go's reference instant is 2006-01-02T15:04:05Z07:00, so
// "2002-10-10T12:00:00-05:00" and "2004-04-12-05:00" matched no field: time.Format
// returned them almost verbatim and the argument was discarded. Every case therefore
// asserts both that the argument came through and that the old literal did not.
func TestConstructorsReflectTheirArgument(t *testing.T) {
	for _, tc := range []struct {
		name    string
		got     string
		want    string
		refuse  []string
		comment string
	}{
		{
			name:    "DateTime",
			got:     string(DateTime("").NewDateTime(reference)),
			want:    "2026-08-31T14:05:06Z",
			refuse:  []string{"2002", "10-10", "12:00:00", "-05:00"},
			comment: `was time.Format("2002-10-10T12:00:00-05:00")`,
		},
		{
			name:    "Date",
			got:     string(Date("").NewDate(reference)),
			want:    "2026-08-31",
			refuse:  []string{"2004", "04-12", "-05:00"},
			comment: `was time.Format("2004-04-12-05:00")`,
		},
		{
			// Not a broken layout, but it was declared on DateTime and returned DateTime,
			// so xsd:time had no constructor while xsd:dateTime had two.
			name:    "Time",
			got:     string(Time("").NewTime(reference)),
			want:    "14:05:06",
			refuse:  nil,
			comment: "was declared on DateTime and returned DateTime",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("= %q, want %q (%s)", tc.got, tc.want, tc.comment)
			}
			for _, dead := range tc.refuse {
				if strings.Contains(tc.got, dead) {
					t.Errorf("= %q, still carries %q from the old layout literal, so the "+
						"argument is being discarded", tc.got, dead)
				}
			}
		})
	}
}

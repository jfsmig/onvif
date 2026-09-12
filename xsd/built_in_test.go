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
	"encoding/xml"
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

// Three validators in this file tested the complement of their own rule, so each returned an
// error for exactly the values it exists to accept and accepted the ones it exists to reject:
//
//   - NewNonNegativeInteger errored when data > 0, so it refused every positive integer and
//     admitted every negative one;
//   - NewPositiveInteger errored when data >= 0, so it refused everything;
//   - NewLanguage errored when the pattern *matched*, and the pattern was unanchored, so
//     regexp.MatchString succeeded on any string containing two letters anywhere.
//
// None of the three has a caller in this repository, so nothing ever reached a camera. That
// is what keeps this a latent defect rather than a wire failure -- and it is also why it went
// unnoticed: they are exported, so the first caller would have inherited the inversion whole.
func TestNumericAndLanguageConstructorsAcceptTheirOwnRange(t *testing.T) {
	t.Run("NonNegativeInteger", func(t *testing.T) {
		for _, ok := range []int64{0, 7} {
			if _, err := NonNegativeInteger(0).NewNonNegativeInteger(ok); err != nil {
				t.Errorf("NewNonNegativeInteger(%d) = %v, want nil", ok, err)
			}
		}
		if _, err := NonNegativeInteger(0).NewNonNegativeInteger(-1); err == nil {
			t.Error("NewNonNegativeInteger(-1) accepted a negative value")
		}
	})

	t.Run("PositiveInteger", func(t *testing.T) {
		if _, err := PositiveInteger(0).NewPositiveInteger(7); err != nil {
			t.Errorf("NewPositiveInteger(7) = %v, want nil", err)
		}
		for _, bad := range []int64{0, -1} {
			if _, err := PositiveInteger(0).NewPositiveInteger(bad); err == nil {
				t.Errorf("NewPositiveInteger(%d) accepted a non-positive value", bad)
			}
		}
	})

	t.Run("Language", func(t *testing.T) {
		for _, ok := range []string{"en", "en-GB", "x-klingon"} {
			if _, err := Language("").NewLanguage(Token(ok)); err != nil {
				t.Errorf("NewLanguage(%q) = %v, want nil", ok, err)
			}
		}
		// Unanchored, the pattern matched the "en" inside these and admitted them whole.
		for _, bad := range []string{"123", "1en2", ""} {
			if _, err := Language("").NewLanguage(Token(bad)); err == nil {
				t.Errorf("NewLanguage(%q) accepted a value that is not a language tag", bad)
			}
		}
	})
}

// TestGregorianConstructorsEmitNumericFields pins five constructors that ignored the format
// written in the doc comment directly above each of them.
//
// Four built their result with fmt.Sprint on a time.Month, whose String method yields the
// English month name, so NewGYearMonth returned "2026-August" where xs:gYearMonth is CCYY-MM.
// NewGDay and NewGYear had the other half of the problem: no zero padding, so the fifth of the
// month was "---5" and the year 500 was "500".
//
// Verified against XML Schema Part 2: section 3.2.10 gYearMonth CCYY-MM, 3.2.11 gYear CCYY,
// 3.2.12 gMonthDay --MM-DD, 3.2.13 gDay ---DD, 3.2.14 gMonth --MM. Every field is a numeral of
// fixed width and none may be left-truncated.
//
// None has a caller in this repository, which is what kept it latent -- and, as with the three
// inverted validators below, is also why it went unnoticed: they are exported, so the first
// caller inherits the defect whole.
func TestGregorianConstructorsEmitNumericFields(t *testing.T) {
	// A year below 1000 and a day below 10, so the padding is what is being measured. The
	// month name would be "March", which no assertion below could mistake for "03".
	padded := time.Date(500, time.March, 5, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name string
		got  string
		want string
	}{
		{"GYearMonth", string(GYearMonth("").NewGYearMonth(reference)), "2026-08"},
		{"GMonthDay", string(GMonthDay("").NewGMonthDay(reference)), "--08-31"},
		{"GMonth", string(GMonth("").NewGMonth(reference)), "--08"},
		{"GDay", string(GDay("").NewGDay(reference)), "---31"},
		{"GYear", string(GYear("").NewGYear(reference)), "2026"},

		{"GYearMonth pads", string(GYearMonth("").NewGYearMonth(padded)), "0500-03"},
		{"GMonthDay pads", string(GMonthDay("").NewGMonthDay(padded)), "--03-05"},
		{"GMonth pads", string(GMonth("").NewGMonth(padded)), "--03"},
		{"GDay pads", string(GDay("").NewGDay(padded)), "---05"},
		{"GYear pads", string(GYear("").NewGYear(padded)), "0500"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("= %q, want %q", tc.got, tc.want)
			}
			if strings.ContainsAny(tc.got, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
				t.Errorf("= %q, still carries a month name: fmt.Sprint is calling "+
					"time.Month.String()", tc.got)
			}
		})
	}
}

// TestGregorianConstructorsPadANegativeYear covers the trap in the fix rather than in the
// defect: %04d counts the sign in its width, so a plain %04d turns year -1 into "-001" where
// the lexical space wants four digits after the sign.
//
// The year numbering itself is deliberately not addressed here -- see the TODO at NewGYear.
func TestGregorianConstructorsPadANegativeYear(t *testing.T) {
	bce := time.Date(-44, time.March, 15, 0, 0, 0, 0, time.UTC)

	if got, want := string(GYear("").NewGYear(bce)), "-0044"; got != want {
		t.Errorf("NewGYear = %q, want %q", got, want)
	}
	if got, want := string(GYearMonth("").NewGYearMonth(bce)), "-0044-03"; got != want {
		t.Errorf("NewGYearMonth = %q, want %q", got, want)
	}
}

// TestStringTypesConstrainOnlyTheirOwnLexicalSpace pins two constructors that rejected
// perfectly legal values.
//
// Both refused any string containing '<', '>' or '&'. Those are ordinary characters in both
// value spaces: XML Schema Part 2 section 3.3.1 defines normalizedString as the strings
// containing no carriage return, line feed or tab, and nothing else, and 3.3.2 adds only the
// space rules for token. Escaping them is the XML writer's job, and encoding/xml already does
// it -- so the constructors were enforcing a serialisation concern one layer too low, and a
// legal ONVIF value such as a URI query string could not be built at all.
//
// NewToken also treated every Unicode space separator as whitespace, through [\s\p{Zs}],
// where 3.3.2 constrains only #x20.
func TestStringTypesConstrainOnlyTheirOwnLexicalSpace(t *testing.T) {
	t.Run("NormalizedString", func(t *testing.T) {
		for _, ok := range []string{"a<b", "a>b", "?a=1&b=2", "plain", "two  spaces", " padded "} {
			if _, err := NormalizedString("").NewNormalizedString(ok); err != nil {
				t.Errorf("NewNormalizedString(%q) = %v, want nil", ok, err)
			}
		}
		// The three that genuinely are outside the value space.
		for _, bad := range []string{"a\rb", "a\nb", "a\tb"} {
			if _, err := NormalizedString("").NewNormalizedString(bad); err == nil {
				t.Errorf("NewNormalizedString(%q) accepted a line break or tab", bad)
			}
		}
	})

	t.Run("Token", func(t *testing.T) {
		// A non-breaking space is not #x20, so token's space rules have nothing to say about
		// it and a value carrying one is legal.
		for _, ok := range []string{"a<b", "?a=1&b=2", "one two", "a b"} {
			if _, err := Token("").NewToken(NormalizedString(ok)); err != nil {
				t.Errorf("NewToken(%q) = %v, want nil", ok, err)
			}
		}
		// Leading, trailing, doubled, and the characters normalizedString already excludes.
		for _, bad := range []string{" a", "a ", "a  b", "a\tb", "a\nb"} {
			if _, err := Token("").NewToken(NormalizedString(bad)); err == nil {
				t.Errorf("NewToken(%q) accepted a value outside the token lexical space", bad)
			}
		}
	})
}

// TestNewLanguageAcceptsTheRecommendationsTags is a second axis on NewLanguage, separate from
// the anchoring and inversion repaired earlier: the pattern itself was the 2001 working
// draft's, which the comment cites honestly.
//
// That pattern admits only a two-letter primary subtag and no digits in any subtag, so "eng",
// "und" and "de-1996" were rejected though they are legal. XML Schema Part 2 section 3.3.3
// gives the pattern as [a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*.
func TestNewLanguageAcceptsTheRecommendationsTags(t *testing.T) {
	for _, ok := range []string{"en", "eng", "und", "de-1996", "en-US-x-1", "x-klingon", "i-navajo"} {
		if _, err := Language("").NewLanguage(Token(ok)); err != nil {
			t.Errorf("NewLanguage(%q) = %v, want nil", ok, err)
		}
	}
	// Still rejected, and the reason the anchors matter: a digit-led tag has no letters to
	// start it, and an over-long subtag is out of range.
	for _, bad := range []string{"123", "1en2", "", "-en", "en-", "abcdefghi"} {
		if _, err := Language("").NewLanguage(Token(bad)); err == nil {
			t.Errorf("NewLanguage(%q) accepted a value that is not a language tag", bad)
		}
	}
}

// TestListTypesMarshalAsOneSpaceSeparatedElement pins the three list datatypes.
//
// XML Schema Part 2 makes NMTOKENS, IDREFS and ENTITIES list types: the lexical space is
// white-space separated tokens inside ONE element. A bare Go slice with no marshaller emits
// one element per item instead, which is a different document.
//
// No field in this repository is of any of these types, so this is entirely for the first
// caller -- and the round trip is asserted, because a type that marshals as a list and
// unmarshals as repeated elements is worse than one that is consistently wrong.
func TestListTypesMarshalAsOneSpaceSeparatedElement(t *testing.T) {
	type doc struct {
		XMLName  xml.Name `xml:"doc"`
		Tokens   NMTOKENS `xml:"tokens"`
		Refs     IDREFS   `xml:"refs"`
		Entities ENTITIES `xml:"entities"`
	}

	in := doc{
		Tokens:   NMTOKENS{"alpha", "beta"},
		Refs:     IDREFS{"r1", "r2"},
		Entities: ENTITIES{"e1"},
	}

	b, err := xml.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	const want = `<doc><tokens>alpha beta</tokens><refs>r1 r2</refs><entities>e1</entities></doc>`
	if got := string(b); got != want {
		t.Errorf("Marshal =\n  %s\nwant\n  %s", got, want)
	}

	var out doc
	if err := xml.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(out.Tokens) != 2 || out.Tokens[0] != "alpha" || out.Tokens[1] != "beta" {
		t.Errorf("round trip lost the list: %#v", out.Tokens)
	}
	if len(out.Entities) != 1 || out.Entities[0] != "e1" {
		t.Errorf("round trip lost a single-item list: %#v", out.Entities)
	}

	// An empty list is an empty element, not a missing one, and must not come back as a
	// slice holding one empty string -- which is what a naive Split on "" yields.
	var empty doc
	if err := xml.Unmarshal([]byte(`<doc><tokens></tokens></doc>`), &empty); err != nil {
		t.Fatalf("Unmarshal empty: %v", err)
	}
	if len(empty.Tokens) != 0 {
		t.Errorf("an empty list unmarshalled to %#v, want no items", empty.Tokens)
	}
}

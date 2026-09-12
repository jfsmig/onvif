// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// Portions of this file derive from the goonvif project, now use-go/onvif,
// originally distributed under the MIT License, see LICENSE.MIT,
// Copyright (c) 2018 Yakovlev Dmitry, Zhorzh Palanjyan, Crazybber.
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

import (
	"errors"
	"regexp"
)

// Duration of iso8601
type Duration struct {
	years  string //= number of years
	months string //= number of months
	days   string //= number of days
	// Time section
	hours   string //= the number of hours
	minutes string //= the number of minutes
	seconds string //= the number of seconds
}

// NewDuration returns a Duration, or an error naming the component that was not a number.
//
// Each component is validated whole. The patterns used to read `^$|[0-9]+`, where alternation
// binds at the top level, so the anchors belonged to the empty branch alone and the digit
// branch had none -- and regexp.MatchString searches rather than matches, so any value holding
// a digit run anywhere was accepted. ISO8601Duration concatenates verbatim, so "12abc" years
// produced "P12abcY". A schema pattern constrains the whole literal, which is what the anchors
// now say.
//
// The lexical space is XML Schema Part 2 section 3.2.6: each component an unsigned integer --
// no sign, the sign of a duration belongs in front of the P -- and only the seconds component
// fractional. An empty string means the component is absent.
//
// The six used to be validated by six copied blocks, four of which named the wrong component
// in their error. A loop cannot drift that way.
func NewDuration(years, months, days, hours, minutes, seconds string) (*Duration, error) {
	for _, c := range []struct {
		name    string
		value   string
		pattern *regexp.Regexp
	}{
		{"years", years, unsignedInteger},
		{"months", months, unsignedInteger},
		{"days", days, unsignedInteger},
		{"hours", hours, unsignedInteger},
		{"minutes", minutes, unsignedInteger},
		{"seconds", seconds, unsignedDecimal},
	} {
		if !c.pattern.MatchString(c.value) {
			return nil, errors.New(c.name + " value = " + c.value +
				" does not match pattern " + c.pattern.String())
		}
	}

	return &Duration{years: years, months: months, hours: hours, days: days, minutes: minutes, seconds: seconds}, nil
}

var (
	// Empty means absent, which is why the quantifier is * rather than + with an alternation.
	unsignedInteger = regexp.MustCompile(`^[0-9]*$`)
	unsignedDecimal = regexp.MustCompile(`^([0-9]+(\.[0-9]+)?)?$`)
)

// ISO8601Duration to string
func (duration Duration) ISO8601Duration() string {
	var result string
	result += "P" // time duration designator
	//years
	if duration.years != "" {
		result += duration.years + "Y"
	}
	if duration.months != "" {
		result += duration.months + "M"
	}
	if duration.days != "" {
		result += duration.days + "D"
	}

	// Any of the three, not all three. The condition read && , so a duration carrying only
	// seconds -- which is what a PullMessages Timeout is (ONVIF Core section 9.1.2) -- skipped
	// the whole time section, fell into the len(result) == 1 branch below and came back as
	// "PT0S". Ten seconds rendered as zero, and xs:duration has no representation of a
	// number of seconds that does not go through here.
	if duration.hours != "" || duration.minutes != "" || duration.seconds != "" {
		result += "T"
		if duration.hours != "" {
			result += duration.hours + "H"
		}
		if duration.minutes != "" {
			result += duration.minutes + "M"
		}
		if duration.seconds != "" {
			result += duration.seconds + "S"
		}
	}

	if len(result) == 1 {
		result += "T0S"
	}

	return result
}

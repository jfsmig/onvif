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

package credentials

// AGENTS.md requires a field holding a password to carry json:"-", and
// xsd/onvif/redaction_test.go pins that rule for the types the CLI dumps. The DTO in
// store.go cannot carry the tag: it exists to be unmarshalled from a JSON file. These
// tests are the compensating control, and they check the three properties that make the
// exception safe -- nothing here marshals a secret, formats one, or wraps one into an
// error.
//
// The error case is the non-obvious one. encoding/json's own message quotes the offending
// character of the document, and if the syntax fault is inside a password string, that
// character is a character of the password. That is why store.go renders decoder failures
// through jsonFault instead of wrapping them.

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jfsmig/onvif/networking"
)

// secret is distinctive enough that finding it anywhere is unambiguous, and shaped so that
// a file built around it is malformed at a byte inside the password.
const secret = "hunter2-SUPER-SECRET"

// numericSecret is the same thing written the one way JSON lets a password appear without
// quotes, which is the shape that reaches UnmarshalTypeError rather than SyntaxError.
const numericSecret = "8675309"

func TestNoErrorMentionsAPassword(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files fstest.MapFS
	}{
		{
			// The quote is unterminated, so the decoder faults inside the password.
			"malformed inside the password",
			fstest.MapFS{"a.json": file(`{"` + camA + `": {"user": "admin", "password": "` + secret + `}}`)},
		},
		{
			"the password is not a string",
			fstest.MapFS{"a.json": file(`{"` + camA + `": {"user": "admin", "password": ["` + secret + `"]}}`)},
		},
		{
			"the whole document is a string",
			fstest.MapFS{"a.json": file(`"` + secret + `"`)},
		},
		{
			// UnmarshalTypeError.Value is "number <literal>" when a numeric literal fails
			// to fit a numeric target. Every field of fileEntry is a string, so this
			// renders without the literal today -- the case is here so that adding an int
			// field, or a json:",string" tag, fails loudly instead of quietly printing an
			// unquoted password.
			"the password is an unquoted number",
			fstest.MapFS{"a.json": file(`{"` + camA + `": {"user": "admin", "password": ` + numericSecret + `}}`)},
		},
		{
			"an entry with no user",
			fstest.MapFS{"a.json": file(`{"` + camA + `": {"passwd": "` + secret + `"}}`)},
		},
		{
			"a duplicate camera",
			fstest.MapFS{
				"a.json": file(jsonFor(camA, "admin", secret)),
				"b.json": file(jsonFor(camA, "admin", secret)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadFS(tc.files)
			if err == nil {
				t.Fatal("want an error")
			}
			for _, leak := range []string{secret, numericSecret} {
				if strings.Contains(err.Error(), leak) {
					t.Fatalf("the password leaked into the error: %v", err)
				}
			}
		})
	}
}

func TestResolverFormattingHidesThePassword(t *testing.T) {
	// fmt prints unexported fields, so the default rendering of any of these would print
	// the secret. AGENTS.md forbids logging a ClientAuth for the same reason; these types
	// hold one, so they carry their own String.
	store, err := LoadFS(fstest.MapFS{"a.json": file(jsonFor(camA, "admin", secret))})
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	static := Static("environment", networking.ClientAuth{Username: "admin", Password: secret})

	for _, tc := range []struct {
		name  string
		value any
	}{
		{"Store", store},
		{"Static", static},
		{"Chain", Chain{store, static}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// %#v is not one more verb of the same kind. It ignores String entirely and
			// prints every unexported field, so it walked straight past the String methods
			// above and printed the whole entry map; only GoString stops it.
			for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
				if got := fmt.Sprintf(verb, tc.value); strings.Contains(got, secret) {
					t.Errorf("%s of a %s printed the password: %s", verb, tc.name, got)
				}
			}
		})
	}
}

func TestStoreDoesNotMarshalToJSON(t *testing.T) {
	// onvif-cli JSON-encodes structs straight to stdout. A Store caught by such a dump
	// must yield nothing, which is what having only unexported fields buys -- and it is
	// the property that replaces the json:"-" tag the DTO cannot carry.
	store, err := LoadFS(fstest.MapFS{"a.json": file(jsonFor(camA, "admin", secret))})
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	b, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(b) != "{}" {
		t.Fatalf("json.Marshal(store) = %s, want {}", b)
	}
}

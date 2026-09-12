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

// Package credentials answers one question: which credentials should be presented to the
// camera bearing this identifier?
//
// A fleet is rarely uniform. The identifier is the device's WS-Discovery endpoint
// reference, so the answer can be kept per camera in $BASEDIR/credentials/*.json and
// looked up by whatever discovery reported, with a blanket credential behind it for the
// fleet that does share one account.
//
// The package sits below sdk on the layering of AGENTS.md: it returns errors and logs
// nothing. Its only dependency in this tree is networking, for ClientAuth, which it reuses
// rather than inventing a second type that carries a password.
//
// Never add a request struct to this package. networking.CallMethod routes a call by the
// last segment of the request struct's PkgPath (networking/client.go:127-131), so a
// request declared here would be addressed to a "credentials" endpoint that no device
// serves. The directory names in this tree are load-bearing; this one is not one of them.
package credentials

import (
	"regexp"
	"strings"

	"github.com/jfsmig/onvif/v2/networking"
)

// Resolver yields the credentials to present to one camera.
//
// The identifier is the device's WS-Discovery endpoint reference, which is what
// wsd.Device.UUID carries. ONVIF Core section 7.1 mandates the urn:uuid form, overriding
// the recommendation of WS-Discovery section 2.6 in favour of "uuid:", and equipment in
// the field emits both; an implementation therefore accepts every spelling, see canonical.
//
// source names where the answer came from -- the file it was read from, or the label a
// Static was built with. It exists because an authentication failure whose credential
// source is unknown is the most confusing outcome this tool produces: "401" and "401 with
// the compiled-in admin/admin" call for different next actions. It never holds a secret.
//
// A miss is reported by ok rather than by an error, because "this resolver knows nothing
// about that camera" is the ordinary outcome that drives a Chain to its next member.
// There is no error return at all: everything that can fail has already failed, or not, by
// the time the Resolver was built.
//
// Resolve must be safe for concurrent use. sdk fans out with sync.WaitGroup.Go and a
// caller may well resolve several cameras at once.
type Resolver interface {
	Resolve(id string) (auth networking.ClientAuth, source string, ok bool)
}

// Static answers the same credentials for every camera, labelled with source.
//
// It is the tail of a Chain: the environment blanket of a fleet that does share one
// account. It always matches, including for an empty ClientAuth, so any member placed
// after it is unreachable.
//
// The concrete type is unexported on purpose. An exported struct holding a password is one
// %v away from a leak, and this one renders as `credentials.Static(environment)`.
func Static(source string, auth networking.ClientAuth) Resolver {
	return staticResolver{source: source, auth: auth}
}

type staticResolver struct {
	source string
	auth   networking.ClientAuth
}

func (s staticResolver) Resolve(string) (networking.ClientAuth, string, bool) {
	return s.auth, s.source, true
}

// String hides the credentials. fmt prints unexported fields, so the default rendering of
// this struct would print the password; AGENTS.md forbids logging a ClientAuth for exactly
// that reason.
func (s staticResolver) String() string { return "credentials.Static(" + s.source + ")" }

// GoString is String. %#v ignores String and prints every unexported field, and fmt
// consults GoString at every nesting depth -- so this one also covers any Chain holding a
// Static, which would otherwise print the password through its member.
func (s staticResolver) GoString() string { return s.String() }

// Chain resolves through its members in order, the first match winning. It is where the
// whole precedence of the tool is written down, and the only place that knows it:
//
//	Chain{homeStore, etcStore, Static("environment", envAuth)}
//
// A member that is a literal nil is skipped. A nil *Store -- what LoadBaseIfPresent returns
// for a base directory nobody created -- is not caught by that check, because an interface
// holding a nil pointer is not nil; what makes it a miss is (*Store).Resolve's own nil
// receiver. A Resolver written without that guard must be kept out of the chain by its
// caller. The zero Chain resolves nothing.
//
// Like a Store, a Chain is assembled before use and never mutated afterwards. That is what
// makes Resolve safe to call from several goroutines: the slice header is read-only by
// then, and appending to a Chain that another goroutine is resolving through is a data
// race like appending to any other slice.
type Chain []Resolver

func (c Chain) Resolve(id string) (networking.ClientAuth, string, bool) {
	for _, r := range c {
		if r == nil {
			continue
		}
		if auth, source, ok := r.Resolve(id); ok {
			return auth, source, true
		}
	}
	return networking.ClientAuth{}, "", false
}

// CanonicalID reduces a camera identifier to the form a Store is keyed by: the endpoint
// reference stripped of its URN prefix and lowercased.
//
// Exported because the same reduction has to be applied wherever two identifiers are
// compared -- an operator's command-line argument against what a device claimed in its
// probe reply, say. Two implementations of this rule would be two chances to disagree.
//
// Both prefixes are stripped because both occur, and both are matched without regard to
// case: ONVIF Core section 7.1 overrides the "uuid:" recommendation of WS-Discovery
// section 2.6 in favour of URN:UUID -- "uuid:" is not a registered URI scheme -- but the
// requirement it forwards to, section 7.3.1, is a "should", so a device spelling it either
// way is conformant and this package has no standing to insist. RFC 8141 section 2.1 makes
// the scheme and the namespace identifier case-insensitive.
//
// What follows the prefix is lowered only when it is a UUID. RFC 4122 section 3 licenses
// exactly that much -- the hexadecimal "is case insensitive on input" -- and no further:
// an endpoint reference is not required to be a UUID (WS-Discovery section 2.6 types the
// address as xs:anyURI), and for anything else RFC 8141 section 3.1 compares the
// namespace-specific string case-sensitively. Folding it regardless would make two devices
// whose identifiers differ only in case collide on one key, which is a duplicate entry at
// load and one camera's password offered to another at lookup.
//
// It normalises and never rejects. The identifier is whatever the device put in its
// EndpointReference/Address, so refusing a value that is not a well-formed UUID would deny
// credentials to precisely the non-conformant camera that needs a hand-written entry.
func CanonicalID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) >= len(urnPrefix) && strings.EqualFold(id[:len(urnPrefix)], urnPrefix) {
		id = id[len(urnPrefix):]
	} else if len(id) >= len(uuidPrefix) && strings.EqualFold(id[:len(uuidPrefix)], uuidPrefix) {
		id = id[len(uuidPrefix):]
	}
	if IsUUID(id) {
		return strings.ToLower(id)
	}
	return id
}

// uuidShape is the 8-4-4-4-12 hexadecimal of RFC 4122 section 3. Anchored, so it matches a
// string that is a UUID and nothing else.
var uuidShape = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a UUID in the textual form of RFC 4122 section 3, with no
// prefix. Exported because the same question is asked outside this package -- a command
// line argument that is a bare UUID is a camera identifier rather than a host name -- and
// two spellings of the rule would be two chances to disagree.
func IsUUID(s string) bool { return uuidShape.MatchString(s) }

const (
	urnPrefix  = "urn:uuid:"
	uuidPrefix = "uuid:"
)

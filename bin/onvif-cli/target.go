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

package main

import (
	"fmt"
	"strings"

	"github.com/jfsmig/onvif/v2/credentials"
)

// deviceTarget is what the single argument of a dump sub-command designates: an address to
// connect to, or the WS-Discovery identifier of a camera to look up first. Exactly one
// field is set.
//
// The two are not interchangeable, and the difference is the whole point of the identifier
// form: credentials are kept per camera under the identifier the device reports, so a
// camera named by its address alone has no identifier and can only be given the blanket
// credential.
type deviceTarget struct {
	Xaddr string // the argument was an address, passed through unchanged
	Uuid  string // canonical "urn:uuid:<lowercase uuid>"
}

// parseDeviceTarget decides what the operator typed.
//
// The rule is deliberately not "does it contain a colon". An IPv6 literal is full of them
// -- "[2001:db8::1]:80" is the shape go-wsd builds from an advertised XAddr -- and would
// have been read as anything but an address. So an identifier is recognised positively and
// everything else is an address passed through untouched: the connect error already names
// the address, and validating here would reject "192.168.1.70" without a port, which works
// today.
//
// A prefixed argument is an identifier whatever follows the prefix, and both prefixes are
// taken. ONVIF Core section 7.1 overrides the "uuid:" recommendation of WS-Discovery
// section 2.6 in favour of URN:UUID, but the requirement it forwards to, section 7.3.1, is
// a "should", and section 2.6 types the address as xs:anyURI -- so a device whose endpoint
// reference is neither spelling nor a UUID is conformant, `discover` prints it in the UUID
// column, and this tool has no standing to refuse the column it just printed. A body that
// no device claims is caught downstream by lookupByUUID, which answers with the list of
// identifiers that did answer -- a better message than a shape rejection.
//
// An unprefixed argument has to be told from a host name, so there the UUID shape is the
// only signal available, and a camera's own web interface showing the bare UUID is why it
// is worth reading. One ambiguity is left, and it has a free way out: a DNS label spelled
// exactly like a UUID is read as an identifier, and adding the port makes it an address
// again, since a UUID never contains a colon.
func parseDeviceTarget(arg string) (deviceTarget, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return deviceTarget{}, fmt.Errorf("empty argument: expected IP:PORT or urn:uuid:<uuid>")
	}

	lower := strings.ToLower(arg)
	for _, prefix := range []string{"urn:uuid:", "uuid:"} {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		if arg[len(prefix):] == "" {
			return deviceTarget{}, fmt.Errorf(
				"invalid device identifier %q: %s must be followed by the identifier, as "+
					"printed in the UUID column of `onvif-cli discover`", arg, prefix)
		}
		return deviceTarget{Uuid: canonicalUUID(arg)}, nil
	}

	if credentials.IsUUID(arg) {
		return deviceTarget{Uuid: canonicalUUID(arg)}, nil
	}
	return deviceTarget{Xaddr: arg}, nil
}

// canonicalUUID spells an identifier the way ONVIF Core section 7.1 asks for, which is the
// form `discover` prints and the form a credentials file is expected to be keyed by.
// credentials.CanonicalID does the reduction, so the argument and the device's own claim
// are compared by one rule rather than two.
func canonicalUUID(id string) string {
	return "urn:uuid:" + credentials.CanonicalID(id)
}

// uuidColumn is the third column of the `discover` and `streams` output. A device that
// reported no endpoint reference prints "-" so that a parser sees a constant number of
// fields.
//
// It is a rendering and nothing else. The placeholder must never travel into a credential
// lookup, where it would name a camera called "-" instead of missing the store the way an
// unknown camera does.
func uuidColumn(uuid string) string {
	if uuid == "" {
		return "-"
	}
	return uuid
}

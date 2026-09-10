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

// `dump` used to take an address and nothing else. It now also takes the WS-Discovery
// identifier of a camera, because that identifier is what a per-camera credentials file is
// keyed by, and an address carries none.
//
// One argument therefore has to be classified, and the obvious rule -- "does it contain a
// colon" -- is wrong: an IPv6 literal is full of them, and "[2001:db8::1]:80" is exactly
// the shape go-wsd builds from an advertised XAddr (wsd/discover.go, deviceOf). So an
// identifier is recognised positively and everything else is an address, which is also
// what keeps "192.168.1.70" without a port working.
//
// Verified against ONVIF Core section 7.1, which overrides the "uuid:" recommendation of
// WS-Discovery section 2.6 in favour of urn:uuid -- go-wsd cites it in wsd/parse.go:44 --
// against section 7.3.1, which makes that a "should" rather than a "shall" and so leaves
// both spellings conformant, and against RFC 4122 section 3 for the case of the hex.

import (
	"testing"

	"github.com/jfsmig/onvif/credentials"
)

const (
	testUUID  = "1419d68a-1dd2-11b2-a105-f44d2e2b1234"
	testUrnID = "urn:uuid:" + testUUID
)

func TestParseDeviceTarget(t *testing.T) {
	for _, tc := range []struct {
		name      string
		arg       string
		wantUuid  string
		wantXaddr string
		wantErr   bool
	}{
		{"the form ONVIF Core 7.1 mandates", testUrnID, testUrnID, "", false},
		{"the URN upper-cased, as firmware ships it", "URN:UUID:" + testUUID, testUrnID, "", false},
		{"upper-case hex, equal per RFC 4122 3", "urn:uuid:1419D68A-1DD2-11B2-A105-F44D2E2B1234", testUrnID, "", false},
		{"the form WS-Discovery 2.6 recommended", "uuid:" + testUUID, testUrnID, "", false},
		{"bare, as a camera's web page shows it", testUUID, testUrnID, "", false},
		{"padded, as a copy-paste leaves it", "  " + testUrnID + "  ", testUrnID, "", false},

		{"a prefix with nothing behind it", "urn:uuid:", "", "", true},
		{"no argument at all", "", "", "", true},

		// A prefixed argument is an identifier whatever follows the prefix. It is not the
		// tool's business to require a UUID there: ONVIF Core section 7.3.1 only says a
		// device *should* use one, WS-Discovery section 2.6 types the address as
		// xs:anyURI, and `discover` prints whatever the device claimed -- so refusing this
		// meant refusing the very column the tool had just printed. A body no device
		// claims is caught by lookupByUUID, which names the ones that did answer.
		{"a prefixed identifier that is not a UUID", "urn:uuid:CAMERA-IN-THE-LOBBY", "urn:uuid:CAMERA-IN-THE-LOBBY", "", false},
		{"a UUID one digit short is still an identifier", "urn:uuid:1419d68a-1dd2-11b2-a105-f44d2e2b123", "urn:uuid:1419d68a-1dd2-11b2-a105-f44d2e2b123", "", false},

		{"an address with a port", "192.168.1.70:8000", "", "192.168.1.70:8000", false},
		{"an address without one, which works today", "192.168.1.70", "", "192.168.1.70", false},
		// The two cases the colon rule would have got wrong, in both directions.
		{"an IPv6 literal, the shape go-wsd produces", "[2001:db8::1]:80", "", "[2001:db8::1]:80", false},
		{"an unbracketed IPv6 address", "2001:db8::1", "", "2001:db8::1", false},
		{"a host name", "camera.lan:80", "", "camera.lan:80", false},
		{"a bare host name", "nvr", "", "nvr", false},
		// A DNS label may legally be spelled like a UUID. Adding the port is the way out,
		// and it costs nothing because a UUID never contains a colon.
		{"a host spelled like a UUID, with its port", testUUID + ":80", "", testUUID + ":80", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDeviceTarget(tc.arg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseDeviceTarget(%q) = %+v, want an error", tc.arg, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDeviceTarget(%q): %v", tc.arg, err)
			}
			if got.Uuid != tc.wantUuid {
				t.Errorf("parseDeviceTarget(%q).Uuid = %q, want %q", tc.arg, got.Uuid, tc.wantUuid)
			}
			if got.Xaddr != tc.wantXaddr {
				t.Errorf("parseDeviceTarget(%q).Xaddr = %q, want %q", tc.arg, got.Xaddr, tc.wantXaddr)
			}
		})
	}
}

func TestParseDeviceTargetAndTheStoreAgreeOnTheIdentifier(t *testing.T) {
	// The argument and the device's own claim have to reduce to the same store key, or a
	// file written from the `discover` output is never found. Both sides go through
	// credentials.CanonicalID for exactly that reason, and this asserts the CLI has not
	// grown a second rule of its own.
	for _, arg := range []string{
		testUrnID,
		"URN:UUID:" + testUUID,
		"uuid:" + testUUID,
		testUUID,
		"urn:uuid:CAMERA-IN-THE-LOBBY",
	} {
		got, err := parseDeviceTarget(arg)
		if err != nil {
			t.Fatalf("parseDeviceTarget(%q): %v", arg, err)
		}
		if credentials.CanonicalID(got.Uuid) != credentials.CanonicalID(arg) {
			t.Errorf("parseDeviceTarget(%q).Uuid = %q, which is not the same camera",
				arg, got.Uuid)
		}
	}
}

func TestParseDeviceTargetSetsExactlyOneField(t *testing.T) {
	// The two fields drive two different paths -- connect, or probe the LAN first -- so a
	// target with both set, or neither, would be a silent mis-route.
	for _, arg := range []string{testUrnID, testUUID, "192.168.1.70:80", "[2001:db8::1]:80", "urn:uuid:ODD"} {
		got, err := parseDeviceTarget(arg)
		if err != nil {
			t.Fatalf("parseDeviceTarget(%q): %v", arg, err)
		}
		if (got.Uuid == "") == (got.Xaddr == "") {
			t.Errorf("parseDeviceTarget(%q) = %+v, want exactly one field set", arg, got)
		}
	}
}

func TestUUIDColumnIsOutputOnly(t *testing.T) {
	// The placeholder keeps the number of fields on a `discover` line constant. It used to
	// be written into the ClientInfo itself, which is harmless while nothing reads that
	// field and becomes a lookup for a camera named "-" the moment something does.
	if got := uuidColumn(""); got != "-" {
		t.Errorf("uuidColumn(\"\") = %q, want \"-\" so a parser sees a constant column count", got)
	}
	if got := uuidColumn(testUrnID); got != testUrnID {
		t.Errorf("uuidColumn(%q) = %q, want it untouched", testUrnID, got)
	}
}

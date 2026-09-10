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

// The identifier that keys a credentials file is whatever the device put in its
// WS-Discovery EndpointReference/Address, and go-wsd copies that text verbatim
// (wsd/parse.go:80). ONVIF Core section 7.1 mandates "urn:uuid:" -- "uuid:" is not a URI
// scheme -- overriding the recommendation of WS-Discovery section 2.6, and equipment ships
// both; RFC 4122 section 3 makes the hex case-insensitive while cameras print it either
// way.
//
// So the same camera can be spelled four ways, and the trap is to canonicalise one side
// only: normalise the query alone and a file written from the `discover` output -- which
// prints the prefixed form -- is never found; normalise the key alone and the prefixed
// value that discovery actually hands us never matches. TestLookupIsNormalisedOnBothSides
// is that bug, pinned.

import (
	"testing"
	"testing/fstest"
)

const bareA = "00000700-0013-0008-0203-ec71db76e907"

func TestCanonicalIDAcceptsEveryFormOfTheEndpointReference(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   string
	}{
		{"the form ONVIF Core 7.1 mandates", "urn:uuid:" + bareA},
		{"the URN scheme upper-cased", "URN:UUID:" + bareA},
		{"the form WS-Discovery 2.6 recommended", "uuid:" + bareA},
		{"that form upper-cased", "UUID:" + bareA},
		{"bare, as a camera's web UI shows it", bareA},
		{"upper-case hex, which RFC 4122 3 makes equal", "urn:uuid:00000700-0013-0008-0203-EC71DB76E907"},
		{"padded, as a hand-edited file can be", "  urn:uuid:" + bareA + "  "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanonicalID(tc.id); got != bareA {
				t.Errorf("CanonicalID(%q) = %q, want %q", tc.id, got, bareA)
			}
		})
	}
}

func TestLookupIsNormalisedOnBothSides(t *testing.T) {
	// Each file is keyed one way and queried the other. Both directions have to work, and
	// a canonicaliser applied to only one of them passes exactly half of these.
	for _, tc := range []struct {
		name    string
		fileKey string
		query   string
	}{
		{"file prefixed, queried bare", "urn:uuid:" + bareA, bareA},
		{"file bare, queried prefixed", bareA, "urn:uuid:" + bareA},
		{"file prefixed, queried upper-case", "urn:uuid:" + bareA, "URN:UUID:" + bareA},
		{"file in the WS-Discovery spelling, queried in the ONVIF one", "uuid:" + bareA, "urn:uuid:" + bareA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, err := LoadFS(fstest.MapFS{"a.json": file(jsonFor(tc.fileKey, "admin", "secret"))})
			if err != nil {
				t.Fatalf("LoadFS: %v", err)
			}
			if _, _, ok := store.Resolve(tc.query); !ok {
				t.Errorf("a file keyed %q was not found by %q", tc.fileKey, tc.query)
			}
		})
	}
}

func TestCanonicalIDDoesNotRejectANonUUIDIdentifier(t *testing.T) {
	// Canonicalisation normalises, it does not validate. An endpoint reference is only
	// required to be a URI -- WS-Discovery section 2.6 types it xs:anyURI, and ONVIF Core
	// section 7.3.1 merely says a device *should* use a URN:UUID -- so refusing one that
	// is not a well-formed UUID would deny credentials to precisely the non-conformant
	// camera that needs a hand-written entry.
	const odd = "urn:uuid:CAMERA-IN-THE-LOBBY"
	store, err := LoadFS(fstest.MapFS{"a.json": file(jsonFor(odd, "admin", "secret"))})
	if err != nil {
		t.Fatalf("LoadFS refused a non-UUID identifier: %v", err)
	}
	// Queried as `discover` would have printed it, prefix and all: what has to survive is
	// the identifier, not a lowercased version of it -- see the test below.
	if _, _, ok := store.Resolve(odd); !ok {
		t.Errorf("a non-UUID identifier did not survive canonicalisation")
	}
}

func TestCanonicalIDFoldsCaseOnlyWhereTheSpecificationsAllow(t *testing.T) {
	// CanonicalID lowercased everything after the prefix. RFC 4122 section 3 licenses that
	// for the hexadecimal alone -- it "is case insensitive on input" -- and the endpoint
	// reference is not required to be a UUID at all. For anything else RFC 8141 section
	// 3.1 compares the namespace-specific string case-sensitively, as does RFC 3986
	// section 6.2.2.1 outside the urn scheme.
	//
	// So two devices whose identifiers differ only in case are two devices, and folding
	// them made one store key: one entry rejected as a duplicate at load, or one camera's
	// password offered to the other at lookup.

	// The prefix folds either way: RFC 8141 section 2.1 makes the scheme and the namespace
	// identifier case-insensitive.
	if CanonicalID("URN:UUID:"+bareA) != CanonicalID("urn:uuid:"+bareA) {
		t.Error("the urn scheme and the uuid namespace identifier must compare case-insensitively")
	}

	// A well-formed UUID still folds, which is the whole of what RFC 4122 section 3 says.
	const upper = "urn:uuid:00000700-0013-0008-0203-EC71DB76E907"
	if got := CanonicalID(upper); got != bareA {
		t.Errorf("CanonicalID(%q) = %q, want %q per RFC 4122 section 3", upper, got, bareA)
	}

	// An identifier that is not a UUID does not: nothing makes its case insignificant.
	if CanonicalID("urn:uuid:CAM-A") == CanonicalID("urn:uuid:cam-a") {
		t.Error("two endpoint references differing only in case were folded into one key; " +
			"RFC 8141 section 3.1 compares the namespace-specific string case-sensitively")
	}
}

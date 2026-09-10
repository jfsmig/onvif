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

// Chain is where the whole precedence of the tool is written down: the file entry for that
// camera first, then the ONVIF_USERNAME / ONVIF_PASSWORD blanket, which a fleet sharing
// one account legitimately sets, then the compiled-in default. Every one of these pins a
// step of that order, because nothing else states it.

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jfsmig/onvif/networking"
)

var blanket = networking.ClientAuth{Username: "fleet", Password: "fleet-password"}

func storeOf(t *testing.T, files fstest.MapFS) *Store {
	t.Helper()
	store, err := LoadFS(files)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	return store
}

func TestChainPrefersTheFileEntryOverTheBlanket(t *testing.T) {
	store := storeOf(t, fstest.MapFS{"a.json": file(jsonFor(camA, "per-camera", "per-camera-password"))})
	chain := Chain{store, Static("environment", blanket)}

	auth, source, ok := chain.Resolve(camA)
	if !ok {
		t.Fatal("the chain resolved nothing for a camera that has a file")
	}
	if auth.Username != "per-camera" {
		t.Errorf("user = %q, want the file entry: a per-camera file beats the blanket", auth.Username)
	}
	if !strings.HasSuffix(source, "a.json") {
		t.Errorf("source = %q, want the file that answered", source)
	}
}

func TestChainFallsThroughToTheBlanket(t *testing.T) {
	// A set ONVIF_USERNAME means "this covers the cameras of this run", so a camera with
	// no file of its own must still be reached, not skipped.
	store := storeOf(t, fstest.MapFS{"a.json": file(jsonFor(camA, "per-camera", "per-camera-password"))})
	chain := Chain{store, Static("environment", blanket)}

	auth, source, ok := chain.Resolve(camB)
	if !ok {
		t.Fatal("a camera with no file got no credentials at all")
	}
	if auth.Username != blanket.Username {
		t.Errorf("user = %q, want the blanket %q", auth.Username, blanket.Username)
	}
	if source != "environment" {
		t.Errorf("source = %q, want %q: the operator has to be able to tell which "+
			"credential was tried", source, "environment")
	}
}

func TestChainSearchesItsStoresInOrder(t *testing.T) {
	// ~/.onvif before /etc/onvif, per camera rather than per directory: a camera the user
	// configured wins, and one only the system knows about is still found.
	home := storeOf(t, fstest.MapFS{"a.json": file(jsonFor(camA, "user-home", "h"))})
	system := storeOf(t, fstest.MapFS{
		"a.json": file(jsonFor(camA, "system", "s")),
		"b.json": file(jsonFor(camB, "system", "s")),
	})
	chain := Chain{home, system, Static("environment", blanket)}

	if auth, _, _ := chain.Resolve(camA); auth.Username != "user-home" {
		t.Errorf("user = %q for a camera both directories name, want the first one", auth.Username)
	}
	if auth, _, _ := chain.Resolve(camB); auth.Username != "system" {
		t.Errorf("user = %q for a camera only the second directory names, want it found there "+
			"-- the directories are chained per camera, not one instead of the other", auth.Username)
	}
}

func TestChainWithNoMatchReportsAMiss(t *testing.T) {
	// Without a Static tail nothing answers, and the miss must stay a miss: an empty
	// ClientAuth sends no UsernameToken at all (networking/wssecurity_test.go), so
	// fabricating one would turn "no credentials" into "deliberately anonymous".
	chain := Chain{storeOf(t, fstest.MapFS{"a.json": file(jsonFor(camA, "admin", "secret"))})}
	if auth, source, ok := chain.Resolve(camB); ok {
		t.Fatalf("Resolve = (%q, %q, true), want a miss", auth.Username, source)
	}
}

func TestStaticAlwaysMatches(t *testing.T) {
	// Including for an empty ClientAuth, which is why a Static is the tail and never the
	// head: anything after it is unreachable.
	if _, source, ok := Static("built-in default", networking.ClientAuth{}).Resolve(camA); !ok || source != "built-in default" {
		t.Errorf("Static did not answer: source=%q ok=%v", source, ok)
	}
}

func TestATypedNilStoreIsAMissAndNotAPanic(t *testing.T) {
	// The CLI builds its Chain from *Store values that may be nil, and an interface holding
	// a nil pointer is NOT nil -- Chain's own `r == nil` guard does not catch one. What
	// makes the call safe is (*Store).Resolve's nil receiver. Both halves are asserted,
	// because the comment on Chain used to claim the first alone, and removing the second
	// would turn an unconfigured host into a panic.
	var absent *Store
	var asResolver Resolver = absent
	if asResolver == nil {
		t.Fatal("a nil *Store compared equal to nil as a Resolver; if that ever becomes " +
			"true, Chain's nil guard is what protects this case and its comment can say so")
	}
	if _, _, ok := (Chain{absent}).Resolve(camA); ok {
		t.Error("a nil *Store answered")
	}
}

func TestChainSkipsNilMembersAndTheZeroChainIsUsable(t *testing.T) {
	// The CLI builds the chain from directories that may not exist, so a nil member is the
	// normal case and must not need a special case at the call site.
	var absent *Store
	chain := Chain{absent, nil, Static("environment", blanket)}
	if _, _, ok := chain.Resolve(camA); !ok {
		t.Error("a nil member broke the chain")
	}
	if _, _, ok := (Chain{}).Resolve(camA); ok {
		t.Error("the zero Chain resolved something")
	}
}

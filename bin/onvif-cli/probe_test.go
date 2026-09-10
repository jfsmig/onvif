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

// The fan-out in probeLAN is the only concurrency this tool has, and until this file
// nothing exercised it: probeLAN reached net.Interfaces and wsd.Discover directly, so
// `go test -race`, which CI runs and which AGENTS.md calls the only automated check the
// concurrency has, never scheduled one of its goroutines. Extracting lookupByUUID gave
// that fan-out a second caller, which is a good moment to give it a first test.
//
// Like interfaces_test.go and signal_test.go, this pins a runtime property rather than a
// wire format.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jfsmig/go-wsd/wsd"
)

// TestFanOutProbesKeepsOneProbePerInterface uses enough interfaces that a shared *itfProbe,
// or a closure that captured the loop variable, loses a result or reports one twice rather
// than happening to work.
func TestFanOutProbesKeepsOneProbePerInterface(t *testing.T) {
	names := make([]string, 64)
	for i := range names {
		names[i] = fmt.Sprintf("itf%02d", i)
	}

	// One interface fails, because discover() and lookupByUUID both branch on probe.err
	// and a shared struct would spread that failure over every interface.
	send := func(_ context.Context, itf string) ([]wsd.Device, error) {
		if itf == names[0] {
			return nil, errors.New("group join refused")
		}
		return []wsd.Device{{Xaddr: itf + ":80", UUID: "urn:uuid:" + itf}}, nil
	}

	probes := fanOutProbes(context.Background(), names, send)

	if len(probes) != len(names) {
		t.Fatalf("len(probes) = %d, want %d", len(probes), len(names))
	}
	for i, probe := range probes {
		if probe.name != names[i] {
			t.Fatalf("probes[%d].name = %q, want %q: the result is documented to keep the "+
				"order net.Interfaces() reported, and discover() prints in that order",
				i, probe.name, names[i])
		}
		if i == 0 {
			if probe.err == nil {
				t.Errorf("probes[0].err was lost; it is what discover() warns on")
			}
			continue
		}
		if probe.err != nil {
			t.Errorf("probes[%d].err = %v, want none: one interface's failure spread to another",
				i, probe.err)
		}
		if len(probe.devices) != 1 || probe.devices[0].UUID != "urn:uuid:"+names[i] {
			t.Errorf("probes[%d] holds another interface's devices: %v", i, probe.devices)
		}
	}
}

func TestFanOutProbesOnAnEmptySet(t *testing.T) {
	// probeLAN returns no probes when every interface was filtered, and both of its
	// callers range over the result without checking it first.
	got := fanOutProbes(context.Background(), nil, func(context.Context, string) ([]wsd.Device, error) {
		t.Error("no interface, so no probe should have been sent")
		return nil, nil
	})
	if len(got) != 0 {
		t.Errorf("len(probes) = %d, want 0", len(got))
	}
}

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

// The signal set main() installs was os.Kill and os.Interrupt. os/signal is explicit that
// "SIGKILL and SIGSTOP may not be caught by a program", so os.Kill was inert and SIGTERM
// -- which is what `kill <pid>`, systemd and `docker stop` send -- was never handled: the
// run kept probing until its own one-minute deadline expired.
//
// Like interfaces_test.go, this pins an operator-facing default rather than a wire format.

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// TestShutdownSignalsAreCatchable pins the set itself. The uncatchable signals are named
// explicitly because listing one looks harmless -- it compiles, it runs, and it silently
// buys nothing.
func TestShutdownSignalsAreCatchable(t *testing.T) {
	uncatchable := map[os.Signal]string{
		os.Kill:         "os.Kill",
		syscall.SIGKILL: "syscall.SIGKILL",
		syscall.SIGSTOP: "syscall.SIGSTOP",
	}

	found := make(map[os.Signal]bool, len(shutdownSignals))
	for _, sig := range shutdownSignals {
		if name, bad := uncatchable[sig]; bad {
			t.Errorf("shutdownSignals contains %s, which os/signal documents as impossible "+
				"to catch, so it can never stop the process", name)
		}
		found[sig] = true
	}

	for _, want := range []struct {
		sig  os.Signal
		name string
		why  string
	}{
		{syscall.SIGTERM, "syscall.SIGTERM", "`kill <pid>`, systemd and a container stop send this"},
		{os.Interrupt, "os.Interrupt", "Ctrl-C at a terminal sends this"},
	} {
		if !found[want.sig] {
			t.Errorf("shutdownSignals is missing %s: %s", want.name, want.why)
		}
	}
}

// TestSIGTERMCancelsTheContext is the behavioural half: it sends the signal to the test
// binary and requires the derived context to be cancelled. A fresh NotifyContext is built
// here rather than reaching into main(), and cancel() is deferred so the handler is
// unregistered and cannot swallow a signal on behalf of a sibling test.
func TestSIGTERMCancelsTheContext(t *testing.T) {
	ctx, cancel := signal.NotifyContext(context.Background(), shutdownSignals...)
	defer cancel()

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("Kill(SIGTERM): %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("SIGTERM did not cancel the context; a graceful stop is not wired up")
	}
}

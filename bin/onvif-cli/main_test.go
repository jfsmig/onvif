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

// The credentials resolver is built once, in the root command's PersistentPreRunE, so that
// a mistyped path or a malformed file stops the run before a camera is contacted. Cobra
// runs only the closest such hook in the chain, so one added on `dump` or on a leaf would
// shadow the root's and leave the resolver nil -- a panic at the first camera instead of
// an error before any of them. Nothing in cobra enforces that, so this does.
//
// Like interfaces_test.go and signal_test.go, it pins an operator-facing property of the
// tool rather than a wire format.

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func TestOnlyTheRootBuildsTheResolver(t *testing.T) {
	root := newRootCommand(context.Background())

	if root.PersistentPreRunE == nil {
		t.Fatal("the root has no PersistentPreRunE, so nothing builds the credentials resolver")
	}

	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			if child.PersistentPreRunE != nil || child.PersistentPreRun != nil {
				t.Errorf("%q defines a persistent pre-run hook, which shadows the root's; "+
					"the resolver would stay nil for its leaves", child.CommandPath())
			}
			walk(child)
		}
	}
	walk(root)
}

func TestBasedirIsOneFlagForTheWholeTool(t *testing.T) {
	// There is exactly one credentials store per process, so every command has to see the
	// same value -- which is what makes this the one flag in the tool that is persistent
	// on the root rather than owned by a command. A leaf redefining it would silently
	// shadow the root's for that leaf alone.
	root := newRootCommand(context.Background())
	if root.PersistentFlags().Lookup("basedir") == nil {
		t.Fatal("--basedir is not a persistent flag of the root, so it reaches no subcommand")
	}

	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			// LocalFlags is the command's own; what the root declares reaches a child
			// through InheritedFlags and must not be redeclared here.
			if child.LocalFlags().Lookup("basedir") != nil {
				t.Errorf("%q declares its own --basedir, shadowing the root's", child.CommandPath())
			}
			walk(child)
		}
	}
	walk(root)
}

func TestEveryDumpLeafDocumentsItsArgument(t *testing.T) {
	// The single positional is the one thing about this tool nobody guesses, and it now
	// has two forms. Every leaf is built by dumpCommand, so an Example that went missing
	// would mean a leaf built by hand somewhere else.
	root := newRootCommand(context.Background())
	var dump *cobra.Command
	for _, child := range root.Commands() {
		if child.Name() == "dump" {
			dump = child
		}
	}
	if dump == nil {
		t.Fatal("no `dump` command")
	}
	for _, leaf := range dump.Commands() {
		if !strings.Contains(leaf.Example, "urn:uuid:") {
			t.Errorf("`dump %s` does not show the identifier form in its Example", leaf.Name())
		}
		if !strings.Contains(leaf.Use, "TARGET") {
			t.Errorf("`dump %s` usage line does not name its argument: %q", leaf.Name(), leaf.Use)
		}
	}
}

func TestVerbosityLevel(t *testing.T) {
	// The default has to be warn and not info: a run in which everything worked prints
	// nothing on stderr, so that `onvif-cli dump all X | jq` is not read beside a running
	// commentary. Warnings and errors are not verbosity and are printed at every count --
	// the first row is what pins that, since anything above warn would hide them.
	for _, tc := range []struct {
		name  string
		count int
		want  zerolog.Level
	}{
		{"no -v: only what went wrong", 0, zerolog.WarnLevel},
		{"-v: what the tool is doing", 1, zerolog.InfoLevel},
		{"-vv: what it loaded and chose", 2, zerolog.DebugLevel},
		{"-vvv: the wire", 3, zerolog.TraceLevel},
		{"more v than levels stays at the last one", 9, zerolog.TraceLevel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := verbosityLevel(tc.count); got != tc.want {
				t.Errorf("verbosityLevel(%d) = %v, want %v", tc.count, got, tc.want)
			}
		})
	}

	// A warning must survive the default, or the credentials-permission warning and the
	// interface whose probe failed are lost on exactly the runs that need them.
	if zerolog.WarnLevel < verbosityLevel(0) {
		t.Error("the default level hides warnings")
	}
}

func TestVerboseIsACountingFlagOnTheRoot(t *testing.T) {
	// -vv only means "more" if the flag counts. Declared as a bool it would be a repeated
	// flag error, and as a string it would want a level name to keep in step with
	// zerolog's.
	root := newRootCommand(context.Background())
	flag := root.PersistentFlags().Lookup("verbose")
	if flag == nil {
		t.Fatal("--verbose is not a persistent flag of the root, so it reaches no subcommand")
	}
	if flag.Shorthand != "v" {
		t.Errorf("--verbose shorthand is %q, want \"v\"", flag.Shorthand)
	}
	if flag.Value.Type() != "count" {
		t.Errorf("--verbose is a %s, want a count so that -vv means more than -v", flag.Value.Type())
	}
}

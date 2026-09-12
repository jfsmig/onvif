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
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
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

// The one-minute deadline used to wrap the whole process, in main(), which was right while
// every command was a request and a reply. `subscribe` is not: a cap there would have ended
// it at one minute, silently and reported as a success, which is a bug that reads as a
// firmware defect on every camera at once.
//
// Written against the AST rather than against behaviour, in the shape of
// sdk/rpc_label_test.go, because the failure it guards against is a copy-paste back into
// main() and nothing else -- there is no run of the binary that would catch it in under a
// minute.
func TestMainDoesNotBoundTheWholeProcess(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	var body *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && fn.Name.Name == "main" {
			body = fn
		}
	}
	if body == nil {
		t.Fatal("no func main in main.go; this test no longer checks what it claims to")
	}

	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if !ok || pkg.Name != "context" {
			return true
		}
		if selector.Sel.Name == "WithTimeout" || selector.Sel.Name == "WithDeadline" {
			t.Errorf("%s: main() bounds the whole process again, so `subscribe` would stop "+
				"after %v and report success; the budget belongs to runOneShot",
				fset.Position(call.Pos()), oneShotDeadline)
		}
		return true
	})
}

func TestSubscribeDocumentsItsTargetsAndTakesNoDiscoveryFlag(t *testing.T) {
	// The positionals are the whole interface of this command, and there is deliberately no
	// --all: unlike `discover`, it authenticates to every target, and a discovery answer is
	// not an authentication -- README.md says as much about `streams -a`, and a long-lived
	// subscription re-authenticates to whatever answered for hours rather than once.
	root := newRootCommand(context.Background())

	var subscribe *cobra.Command
	for _, child := range root.Commands() {
		if child.Name() == "subscribe" {
			subscribe = child
		}
	}
	if subscribe == nil {
		t.Fatal("no `subscribe` command")
	}

	if !strings.Contains(subscribe.Use, "TARGET [TARGET...]") {
		t.Errorf("the usage line does not say that several cameras may be named: %q", subscribe.Use)
	}
	for _, want := range []string{"urn:uuid:", "192.168.1.70:80"} {
		if !strings.Contains(subscribe.Example, want) {
			t.Errorf("the Example does not show the %s target form", want)
		}
	}
	if subscribe.Flags().Lookup("all") != nil {
		t.Error("`subscribe` declares --all, which would send the resolved credentials to " +
			"whatever answered discovery, for as long as the run lasts")
	}

	// At least one target, and no upper bound: the fleet form is the point of the command,
	// and no argument does not mean "every camera".
	if err := subscribe.Args(subscribe, nil); err == nil {
		t.Error("`subscribe` with no target is accepted; it would subscribe to nothing")
	}
	if err := subscribe.Args(subscribe, []string{"a", "b", "c"}); err != nil {
		t.Errorf("`subscribe` refuses three targets: %v", err)
	}
}

// `subscribe` writes to stdout synchronously under a mutex, so an Encode blocked on a pipe
// nobody is reading is not interruptible by a context: no goroutine is in a select to notice
// the first signal at all, and `onvif-cli subscribe … | less` left unread hangs. Since
// signal.NotifyContext keeps its handler registered after it fires, the second Ctrl-C is
// swallowed too and only SIGKILL ends the run.
//
// context.AfterFunc(ctx, stop) restores the default disposition, so the second signal takes
// what the first asked for. Verified by hand against a program with and without it: without,
// the second SIGINT is swallowed and the process sleeps it out; with, it exits 130.
//
// Pinned by reading main(), in the shape of TestMainDoesNotBoundTheWholeProcess, because
// there is no way to assert it in-process — a test that proved it would kill the test binary.
func TestMainLetsASecondSignalThrough(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	var body *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && fn.Name.Name == "main" {
			body = fn
		}
	}
	if body == nil {
		t.Fatal("no func main in main.go; this test no longer checks what it claims to")
	}

	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if ok && pkg.Name == "context" && selector.Sel.Name == "AfterFunc" {
			found = true
		}
		return true
	})
	if !found {
		t.Error("main() no longer arms a second-signal escape, so a `subscribe` whose stdout " +
			"has stopped draining can only be ended with SIGKILL")
	}
}

// A bare `onvif-cli`, and a bare `onvif-cli dump`, used to print one line: the fatal
// "missing sub-command", with no hint that discover, streams, subscribe or dump exist. That
// is the first thing an operator types, before they have learnt that --help is there.
//
// Mistyping a subcommand was always handled well -- cobra answers `dump Ptz` with
// `unknown command "Ptz"` and the whole usage block -- which is the proof that the usage
// block is what helps here. SilenceUsage is set unconditionally in PersistentPreRunE on the
// reasoning that parsing has already succeeded, and that is right for every other error;
// a missing subcommand is the one case where the usage block *is* the message.
//
// It goes to stderr, not stdout: these commands print machine-parsable output, and a usage
// block in the middle of it would be indistinguishable from data to whatever is reading.
func TestABareInvocationSaysWhichCommandsExist(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"root", []string{}},
		{"dump", []string{"dump"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := newRootCommand(context.Background())
			var out, errOut bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&errOut)
			root.SetArgs(tc.args)

			if err := root.Execute(); err == nil {
				t.Fatal("a missing sub-command is still an error, and must stay one")
			}
			if got := errOut.String(); !strings.Contains(got, "Available Commands:") {
				t.Errorf("nothing on stderr lists the commands:\n%s", got)
			}
			if out.Len() != 0 {
				t.Errorf("the usage block reached stdout, where data belongs: %q", out.String())
			}
		})
	}
}

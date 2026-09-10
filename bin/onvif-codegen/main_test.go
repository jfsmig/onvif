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

// This generator produces 206 of the repository's 248 .go files and had no tests at all.
//
// Only the input side is covered here. The shape of the generated output is already pinned
// harder than a test could manage: CI runs `go generate ./...` followed by
// `git diff --quiet --exit-code`, which compares all 206 files byte for byte against what
// is committed. What that gate cannot see is a malformed calls.txt being accepted, because
// the guards below are what stop such a line from ever reaching the template -- and one of
// them, the identifier check, is what keeps a "../" entry from writing outside the package.

import (
	"os"
	"path/filepath"
	"testing"
)

// callsFile writes a calls.txt with the given content and returns its path.
func callsFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "calls.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// TestGetMethodsAcceptsTheDocumentedSyntax covers every form AGENTS.md and the parser's own
// comments say a calls.txt line may take.
func TestGetMethodsAcceptsTheDocumentedSyntax(t *testing.T) {
	// Deliberately includes a file with no trailing newline, which is how all four
	// calls.txt in this repository are actually stored.
	const content = "# a comment\n" +
		"\n" +
		"GetProfiles\n" +
		"  GetCapabilities  \n" +
		"GetStreamUri # a trailing comment\n" +
		"   # an indented comment\n" +
		"SystemReboot"

	methods, err := getMethods(callsFile(t, content))
	if err != nil {
		t.Fatalf("getMethods: %v", err)
	}

	want := []string{"GetProfiles", "GetCapabilities", "GetStreamUri", "SystemReboot"}
	if len(methods) != len(want) {
		t.Fatalf("got %d methods, want %d: %v", len(methods), len(want), methods)
	}
	for i, name := range want {
		if methods[i].Name != name {
			t.Errorf("method %d = %q, want %q", i, methods[i].Name, name)
		}
	}
}

// TestGetMethodsRejectsAnUnusableName is the guard that matters. The entry becomes both a
// Go type reference and a file name, so a name that is not an exported identifier yields
// either a file that cannot compile or -- with a slash or ".." -- a write outside the
// package directory.
func TestGetMethodsRejectsAnUnusableName(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		why     string
	}{
		{"unexported", "getProfiles", "an unexported name cannot reference the request type"},
		{"not an identifier", "Get-Profiles", "a hyphen is not valid in a Go identifier"},
		{"parent directory", "../escape", "would write outside the package directory"},
		{"path separator", "a/b", "would write outside the package directory"},
		{"empty file", "", "a package with no operations is a mistake, not an empty result"},
		{"comments only", "# nothing here\n\n# still nothing", "same, once comments are stripped"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := getMethods(callsFile(t, tc.content)); err == nil {
				t.Errorf("getMethods accepted %q: %s", tc.content, tc.why)
			}
		})
	}
}

func TestGetMethodsReportsAMissingFile(t *testing.T) {
	if _, err := getMethods(filepath.Join(t.TempDir(), "absent.txt")); err == nil {
		t.Error("getMethods accepted a path that does not exist")
	}
}

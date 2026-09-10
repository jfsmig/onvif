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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOutputDirRequiresAMatchingPackageDirectory pins the check that keeps the generator
// writing where it is supposed to.
//
// Without it, `onvif-codegen sdk device somewhere/else/calls.txt` emitted a whole package
// into whatever directory held the source file -- and CI stayed green, because
// `git diff --quiet` does not report untracked files. The mismatch has to be an error at
// generation time or nothing catches it at all.
func TestOutputDirRequiresAMatchingPackageDirectory(t *testing.T) {
	root := t.TempDir()

	matching := filepath.Join(root, "device")
	if err := os.Mkdir(matching, 0o750); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Run("directory matches the package", func(t *testing.T) {
		dir, err := outputDir("device", filepath.Join(matching, "calls.txt"))
		if err != nil {
			t.Fatalf("outputDir: %v", err)
		}
		if dir != matching {
			t.Errorf("outputDir = %q, want %q", dir, matching)
		}
	})

	t.Run("directory does not match the package", func(t *testing.T) {
		_, err := outputDir("media", filepath.Join(matching, "calls.txt"))
		if err == nil {
			t.Fatal("outputDir accepted package \"media\" for a directory named \"device\"; " +
				"the generator would write a media package into device/")
		}
		// The message has to name both, or the operator cannot tell which half is wrong.
		for _, want := range []string{"media", "device"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not mention %q", err, want)
			}
		}
	})
}

// TestReportOrphansNamesTheStaleFile pins the other half of the same problem. Removing a
// line from calls.txt leaves its _auto.go behind, and `git diff` sees no change because the
// file's content did not change -- so the stale wrapper would keep compiling forever. The
// generator has to fail and say which file, and must not delete anything itself.
func TestReportOrphansNamesTheStaleFile(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{
		"GetProfiles" + generatedSuffix,
		"GetStreamUri" + generatedSuffix,
		"types.go", // not generated: must be ignored
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package x\n"), 0o600); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	t.Run("every generated file has an entry", func(t *testing.T) {
		err := reportOrphans(dir, []Method{{Name: "GetProfiles"}, {Name: "GetStreamUri"}})
		if err != nil {
			t.Errorf("reportOrphans: %v", err)
		}
	})

	t.Run("a generated file has no entry", func(t *testing.T) {
		err := reportOrphans(dir, []Method{{Name: "GetProfiles"}})
		if err == nil {
			t.Fatal("reportOrphans accepted a stale GetStreamUri_auto.go")
		}
		if !strings.Contains(err.Error(), "GetStreamUri"+generatedSuffix) {
			t.Errorf("error %q does not name the stale file", err)
		}
		// Reporting, not deleting: the generator must not remove a file the operator may
		// have meant to keep.
		if _, statErr := os.Stat(filepath.Join(dir, "GetStreamUri"+generatedSuffix)); statErr != nil {
			t.Errorf("reportOrphans removed the file it reported: %v", statErr)
		}
	})

	t.Run("hand-written files are left alone", func(t *testing.T) {
		// types.go is not *_auto.go, so it must never be reported however short the
		// method list is.
		err := reportOrphans(dir, []Method{{Name: "GetProfiles"}, {Name: "GetStreamUri"}})
		if err != nil && strings.Contains(err.Error(), "types.go") {
			t.Errorf("reportOrphans reported the hand-written types.go: %v", err)
		}
	})
}

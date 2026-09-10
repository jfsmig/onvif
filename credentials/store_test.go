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

// The one credential the CLI had went to every camera of a run, so a fleet that does not
// share an account could not be expressed at all. These pin the loader that replaced it.
//
// The format is the one found in ~/.onvif/credentials on the development host: a JSON
// object mapping a camera identifier to a "user" and a "password". The map key identifies
// the camera, not the file name -- so these tests deliberately name files after nothing in
// particular, which is the property a "one file per uuid" reading would quietly break.
//
// The loader is exercised over an fs.FS rather than a directory. No camera and no
// credentials file exists in CI, so a suite that needed either would not run there.

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

const (
	camA = "urn:uuid:00000700-0013-0008-0203-ec71db76e907"
	camB = "urn:uuid:04151400-0600-0000-0508-ec71db1a6512"
	camC = "urn:uuid:08010013-1111-0206-0800-ec71dbe17a55"
)

// jsonFor builds the content of a credentials file holding one camera.
func jsonFor(id, user, password string) string {
	return `{"` + id + `": {"user": "` + user + `", "password": "` + password + `"}}`
}

// file is a 0600 regular file, the mode the documentation asks operators for.
func file(content string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(content), Mode: 0o600}
}

func TestLoadReadsEveryJSONFileInTheDirectory(t *testing.T) {
	// The three entries are spread over three files, and the directory also holds what an
	// operator's credentials directory really holds: a note, an editor backup, and a
	// subdirectory. Only the *.json regular files may contribute, and none of the rest may
	// turn into an error -- a stray README must not stop the tool.
	fsys := fstest.MapFS{
		"a.json":        file(jsonFor(camA, "admin", "pass-a")),
		"b.json":        file(jsonFor(camB, "operator", "pass-b")),
		"c.json":        file(jsonFor(camC, "viewer", "pass-c")),
		"README":        file("these are the bench cameras"),
		"a.json~":       file(jsonFor(camA, "stale", "stale")),
		"old/keep.json": file(jsonFor(camA, "older", "older")),
	}

	store, err := LoadFS(fsys)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if store.Len() != 3 {
		t.Fatalf("Len() = %d, want 3: only the *.json files of the directory itself count", store.Len())
	}
	for _, tc := range []struct{ id, user, password string }{
		{camA, "admin", "pass-a"},
		{camB, "operator", "pass-b"},
		{camC, "viewer", "pass-c"},
	} {
		auth, source, ok := store.Resolve(tc.id)
		if !ok {
			t.Fatalf("Resolve(%s) missed", tc.id)
		}
		if auth.Username != tc.user || auth.Password != tc.password {
			t.Errorf("Resolve(%s) returned the wrong entry, user=%q", tc.id, auth.Username)
		}
		if source == "" {
			t.Errorf("Resolve(%s) reported no source; an operator debugging a 401 needs "+
				"to know which file answered", tc.id)
		}
	}
}

func TestLoadAcceptsSeveralEntriesInOneFile(t *testing.T) {
	// The format is a map, so one file may hold a whole fleet. The dev host writes one
	// entry per file, which is a convention and not the format.
	fsys := fstest.MapFS{
		"fleet.json": file(`{
			"` + camA + `": {"user": "admin", "password": "one"},
			"` + camB + `": {"user": "admin", "password": "two"}
		}`),
	}

	store, err := LoadFS(fsys)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if store.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", store.Len())
	}
	if auth, _, _ := store.Resolve(camB); auth.Password != "two" {
		t.Errorf("the second entry of the file was not loaded")
	}
}

func TestLoadIgnoresTheFileName(t *testing.T) {
	// The key inside identifies the camera. Keying on the file name instead would look
	// right on the dev host, where every file is named after its uuid, and would break the
	// moment someone renamed one or put two cameras in a file.
	fsys := fstest.MapFS{"anything-at-all.json": file(jsonFor(camA, "admin", "secret"))}

	store, err := LoadFS(fsys)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if _, _, ok := store.Resolve(camA); !ok {
		t.Fatalf("Resolve(%s) missed: the file name is decoration, the JSON key is the identity", camA)
	}
}

func TestResolveMissesUnknownCamera(t *testing.T) {
	// A miss must be a miss. Returning an empty ClientAuth with ok=true would send an
	// empty username, which networking/wssecurity_test.go pins as "no UsernameToken at
	// all" -- an unauthenticated request rather than a failed lookup.
	store, err := LoadFS(fstest.MapFS{"a.json": file(jsonFor(camA, "admin", "secret"))})
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if auth, source, ok := store.Resolve(camB); ok {
		t.Fatalf("Resolve(%s) = (%q, %q, true), want a miss", camB, auth.Username, source)
	}
}

func TestLoadAcceptsAnEmptyDirectory(t *testing.T) {
	// A directory that exists and holds nothing says "nothing is configured here", which
	// is a usable answer and not a failure.
	store, err := LoadFS(fstest.MapFS{"README": file("nothing yet")})
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if store == nil || store.Len() != 0 {
		t.Fatalf("want a usable empty store, got %v", store)
	}
}

func TestLaxPermissionsReportsWithoutRefusing(t *testing.T) {
	// Every file in ~/.onvif/credentials on the development host is 0664. Refusing them
	// the way ssh refuses a world-readable private key would break the feature on the day
	// it ships, and this package may not log, so it reports and loads.
	fsys := fstest.MapFS{
		"open.json":   {Data: []byte(jsonFor(camA, "admin", "a")), Mode: 0o664},
		"closed.json": {Data: []byte(jsonFor(camB, "admin", "b")), Mode: 0o600},
	}

	store, err := LoadFS(fsys)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if got := store.LaxPermissions(); len(got) != 1 || got[0] != "open.json" {
		t.Errorf("LaxPermissions() = %v, want [open.json]", got)
	}
	if store.Len() != 2 {
		t.Errorf("Len() = %d, want 2: a lax mode is reported, never a refusal", store.Len())
	}
}

func TestInsecureMode(t *testing.T) {
	for _, tc := range []struct {
		mode fs.FileMode
		want bool
	}{
		{0o600, false},
		{0o400, false},
		{0o700, false},
		{0o640, true}, // the group can read it
		{0o644, true}, // anyone can read it
		{0o666, true},
		{0o604, true},
	} {
		if got := InsecureMode(tc.mode); got != tc.want {
			t.Errorf("InsecureMode(%v) = %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestLoadOnTheRealFilesystem(t *testing.T) {
	// The os.DirFS wiring is the one thing an fstest-only suite cannot see: a slip in the
	// rooting would pass every test above and read nothing on a real host.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(jsonFor(camA, "admin", "secret")), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("Load(%s): %v", dir, err)
	}
	auth, source, ok := store.Resolve(camA)
	if !ok || auth.Username != "admin" {
		t.Fatalf("Resolve(%s) = (%q, %v), want the entry of the file", camA, auth.Username, ok)
	}
	// The source has to be openable. A bare "a.json" would send an operator looking in
	// their working directory.
	if want := filepath.Join(dir, "a.json"); source != want {
		t.Errorf("source = %q, want the full path %q", source, want)
	}
}

func TestLoadOnAMissingDirectoryIsErrNotExist(t *testing.T) {
	// This is what lets the tool tell a mistyped --basedir from a corrupt file, and report
	// the first while tolerating a default location nobody created.
	_, err := Load(filepath.Join(t.TempDir(), "absent"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load(absent) = %v, want it to match fs.ErrNotExist", err)
	}
}

func TestLoadBaseIfPresentToleratesAbsence(t *testing.T) {
	// A host that never configured credentials is the normal case, not a failure. The nil
	// store is what Chain skips.
	store, err := LoadBaseIfPresent(filepath.Join(t.TempDir(), "no-such-base"))
	if err != nil {
		t.Fatalf("LoadBaseIfPresent on an absent base = %v, want no error", err)
	}
	if store != nil {
		t.Fatalf("want a nil store for an absent base, got %v", store)
	}
	// And a nil *Store must stay usable, because Chain hands it to Resolve.
	if _, _, ok := store.Resolve(camA); ok {
		t.Errorf("a nil store resolved something")
	}
}

func TestLoadBaseReadsTheCredentialsSubDirectory(t *testing.T) {
	// The documented layout is $BASEDIR/credentials/*.json. Reading $BASEDIR itself would
	// work on a hand-made test directory and find nothing on a real host.
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, SubDir), 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, SubDir, "a.json"),
		[]byte(jsonFor(camA, "admin", "secret")), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store, err := LoadBaseIfPresent(base)
	if err != nil {
		t.Fatalf("LoadBaseIfPresent: %v", err)
	}
	if store == nil {
		t.Fatal("LoadBaseIfPresent returned no store for a base that has a credentials directory")
	}
	if _, _, ok := store.Resolve(camA); !ok {
		t.Errorf("the entry under %s/%s was not read", base, SubDir)
	}
}

// A credentials directory is a place where operators keep things. fstest.MapFS covers the
// stray README and the subdirectory above, but can express neither a symbolic link nor a
// fifo -- and those are the two entries that actually misbehaved -- so these run against a
// real directory.
func TestLoadSkipsWhatIsNotARegularFile(t *testing.T) {
	t.Run("a fifo does not hang the loader", func(t *testing.T) {
		// Reading a fifo blocks until a writer appears. The loader runs in the CLI's
		// PersistentPreRunE, which has no context and no timeout, so the tool would hang
		// before contacting a camera and before printing anything at all.
		dir := t.TempDir()
		if err := exec.Command("mkfifo", filepath.Join(dir, "a.json")).Run(); err != nil {
			t.Skip("mkfifo unavailable:", err)
		}
		done := make(chan error, 1)
		go func() { _, err := Load(dir); done <- err }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Load blocked on a fifo named *.json, with no context to bound it")
		}
	})

	t.Run("a symlinked directory is skipped rather than fatal", func(t *testing.T) {
		// It used to abort the whole load with "is a directory", losing every other camera
		// in the directory along with it.
		dir := t.TempDir()
		if err := os.Symlink(t.TempDir(), filepath.Join(dir, "cam.json")); err != nil {
			t.Fatalf("Symlink: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "real.json"),
			[]byte(jsonFor(camA, "admin", "secret")), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		store, err := Load(dir)
		if err != nil {
			t.Fatalf("a symlinked directory aborted the load: %v", err)
		}
		if _, _, ok := store.Resolve(camA); !ok {
			t.Error("the sibling entry was lost with it")
		}
	})
}

func TestLaxPermissionsFollowsASymlinkToItsTarget(t *testing.T) {
	// os.DirFS reports a DirEntry whose Info is an lstat, so a credentials file kept as a
	// symlink into /run/secrets -- an ordinary layout -- was judged by the mode of the
	// link, which is 0777 on Linux. The tool then warned "consider chmod 600" on every run
	// about a file that was already 0600, and a warning nobody believes protects nothing.
	dir, secrets := t.TempDir(), t.TempDir()
	target := filepath.Join(secrets, "cam.json")
	if err := os.WriteFile(target, []byte(jsonFor(camA, "admin", "secret")), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "cam.json")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if lax := store.LaxPermissions(); len(lax) != 0 {
		t.Errorf("LaxPermissions() = %v for a 0600 target reached through a symlink; the "+
			"mode that matters is the one of the file holding the password", lax)
	}
	if _, _, ok := store.Resolve(camA); !ok {
		t.Error("the entry behind the symlink was not read")
	}
}

func TestLoadRefusesAnOversizedFile(t *testing.T) {
	// A credentials file is a few hundred bytes written by hand. Without the cap a stray
	// large file named *.json is read whole into memory at start-up; with it, the run stops
	// with a message naming the file.
	dir := t.TempDir()
	name := filepath.Join(dir, "huge.json")
	if err := os.WriteFile(name, []byte(strings.Repeat("x", MaxFileBytes+1)), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := Load(dir)
	if !errors.Is(err, ErrMalformedFile) {
		t.Fatalf("err = %v, want it to match ErrMalformedFile", err)
	}
	if !strings.Contains(err.Error(), "huge.json") {
		t.Errorf("err = %v, want the file named", err)
	}
}

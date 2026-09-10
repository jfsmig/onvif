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

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jfsmig/onvif/networking"
)

// SubDir is the directory, under a base directory, that holds the credential files.
// Exported so that the tool and this package cannot disagree about it by a typo.
const SubDir = "credentials"

// fileEntry is the on-disk shape of one camera's credentials.
//
// AGENTS.md requires a field carrying a password to be tagged json:"-", and
// xsd/onvif/redaction_test.go pins that rule. This field cannot carry the tag: it exists
// precisely to be unmarshalled from JSON. The rule protects one thing -- that a secret
// never reaches a dump on stdout -- and three properties give that here instead:
//
//   - the type is unexported, so it cannot appear inside any struct a caller marshals;
//   - it lives only inside load, which converts it to a networking.ClientAuth and drops it;
//   - it is never formatted, never given a String method, and never wrapped into an error.
//
// credentials/redaction_test.go pins the two of those a test can reach: that no error the
// loader produces contains a password, and that nothing holding one marshals or formats
// it. The third is a rule about the body of this file that no test can enforce -- keep
// fileEntry out of every format string.
type fileEntry struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// entry is one camera's credentials and the file they were read from. The file name is
// kept so that Resolve can report its source: an operator debugging a 401 needs to know
// which file answered, and the path is not a secret.
type entry struct {
	auth networking.ClientAuth
	file string
}

// Store is a set of camera credentials read once from a directory of JSON files.
//
// Each *.json file in the directory is a JSON object mapping a camera identifier to an
// object with "user" and "password". One file per camera is the usual layout, and the file
// is then named after the UUID, but the name is decoration: only the keys inside identify
// a camera, so a renamed file keeps working and one file may hold a whole fleet.
//
// A Store is immutable once it has been loaded, and that is its entire concurrency story:
// there is no lock on Resolve because there is nothing to lock. Nothing here writes to
// disk.
type Store struct {
	byID map[string]entry // keyed by CanonicalID(id)
	lax  []string         // files whose mode grants access to group or other
}

// Load reads the *.json files of dir, which is the directory that directly contains them
// -- typically filepath.Join(base, SubDir).
//
// A missing dir is an error wrapping fs.ErrNotExist. Load is the strict entry point: it is
// what an operator who named a directory should get, because a typo that silently loaded
// nothing would surface as an unexplained 401 on every camera. Use LoadBaseIfPresent for a
// default location, where absence is normal.
func Load(dir string) (*Store, error) {
	store, err := LoadFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	// The fs.FS is rooted at dir, so the names it reported are bare. Qualify them once,
	// here, where the real path is still known: everything downstream reports a source an
	// operator can open.
	for id, e := range store.byID {
		e.file = filepath.Join(dir, e.file)
		store.byID[id] = e
	}
	for i, name := range store.lax {
		store.lax[i] = filepath.Join(dir, name)
	}
	return store, nil
}

// LoadBase is Load over base/credentials.
func LoadBase(base string) (*Store, error) { return Load(filepath.Join(base, SubDir)) }

// LoadBaseIfPresent returns a nil Store and a nil error when base/credentials does not
// exist, so that a default location nobody created costs the caller no special case: Chain
// skips a nil member. Every other failure is still an error -- a directory that exists but
// cannot be read is a misconfiguration, not an absence.
func LoadBaseIfPresent(base string) (*Store, error) {
	store, err := LoadBase(base)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return store, nil
}

// LoadFS is Load over an arbitrary filesystem, so that the loader can be exercised without
// touching the real one -- which matters here, because no camera and no credentials file
// exists in CI. Load is os.DirFS plus LoadFS and nothing else.
func LoadFS(fsys fs.FS) (*Store, error) {
	names, lax, err := credentialFiles(fsys)
	if err != nil {
		return nil, err
	}

	store := &Store{byID: make(map[string]entry, len(names)), lax: lax}
	for _, name := range names {
		if err := store.loadFile(fsys, name); err != nil {
			return nil, err
		}
	}
	return store, nil
}

// credentialFiles lists the *.json files of fsys, in directory order, along with those
// whose mode grants access beyond the owner.
//
// fs.ReadDir rather than fs.Glob: Glob "ignores file system errors such as I/O errors
// reading directories", which is exactly the failure that must not pass for an empty
// directory. Anything that is not a *.json regular file is skipped without a word --
// a credentials directory is a place where operators keep notes and an editor keeps
// backups.
func credentialFiles(fsys fs.FS) (names, lax []string, err error) {
	dir, err := fs.ReadDir(fsys, ".")
	if err != nil {
		// The path in that error is always ".", since the filesystem is rooted at the
		// directory being read. Only the caller knows the name worth printing, so the
		// operation and the useless "." are dropped and the cause is returned bare --
		// still matching fs.ErrNotExist and fs.ErrPermission, which is what a caller
		// tests.
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			return nil, nil, pathErr.Err
		}
		return nil, nil, err
	}
	for _, item := range dir {
		if !strings.HasSuffix(item.Name(), ".json") {
			continue
		}
		// fs.Stat rather than item.Info(): os.DirFS reports a DirEntry whose Info is an
		// lstat, so a credentials file kept as a symlink into /run/secrets -- an ordinary
		// layout -- was judged by the mode of the link, 0777 on Linux, while the mode of
		// the file actually holding the password was never looked at. The check then
		// warned on every run about a file that was already 0600, and a warning nobody
		// believes protects nothing.
		info, err := fs.Stat(fsys, item.Name())
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", item.Name(), err)
		}
		// Only a regular file, which is what this function has always claimed to take.
		// Opening anything else is at best an error -- a symlink to a directory aborted
		// the whole load, losing every other camera in it -- and at worst a hang: reading
		// a fifo named "a.json" blocks until a writer appears, and this loader runs before
		// any context exists to bound it.
		if !info.Mode().IsRegular() {
			continue
		}
		// A credentials file is a few hundred bytes written by hand. The cap is the local
		// counterpart of networking.MaxResponseBytes: it costs one comparison, the size is
		// already in hand, and it turns a stray huge file from an out-of-memory at start-up
		// into a message naming it.
		if info.Size() > MaxFileBytes {
			return nil, nil, fmt.Errorf("%w: %s: %d bytes, more than the %d a credentials file may hold",
				ErrMalformedFile, item.Name(), info.Size(), MaxFileBytes)
		}
		names = append(names, item.Name())
		if InsecureMode(info.Mode()) {
			lax = append(lax, item.Name())
		}
	}
	return names, lax, nil
}

// MaxFileBytes bounds one credentials file. Hand-written files are a few hundred bytes;
// the limit exists so that a stray large file named *.json fails with a message instead of
// being read whole into memory, as networking.MaxResponseBytes does on the wire.
const MaxFileBytes = 1 << 20

// InsecureMode reports a mode that lets another local account read the file. It is
// exported because the decision of what to do about it belongs to the caller: this package
// may not log, and refusing to load the way ssh refuses a world-readable private key would
// be a silent failure with no channel to explain itself.
func InsecureMode(mode fs.FileMode) bool { return mode.Perm()&0o077 != 0 }

// loadFile adds the entries of one file to the store.
//
// The first failure aborts the whole load rather than skipping the file. A credentials
// directory is small and hand-maintained, so a store silently missing one camera would
// reproduce exactly the unexplained 401 this package exists to remove. This is not the
// "sdk swallows per-call errors" case of AGENTS.md: that convention exists for camera
// variability, which the operator cannot fix, whereas a JSON typo is their own file.
func (s *Store) loadFile(fsys fs.FS, name string) error {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		// A truncated write, not an intention: "{}" is how a file says it holds nothing.
		// The package cannot log, so an error is the only channel by which the operator
		// learns that a file they wrote does nothing.
		return fmt.Errorf("%w: %s: the file is empty", ErrMalformedFile, name)
	}

	var entries map[string]fileEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return fmt.Errorf("%w: %s: %s", ErrMalformedFile, name, jsonFault(err))
	}

	for id, fe := range entries {
		key := CanonicalID(id)
		if key == "" {
			return fmt.Errorf("%w: %s: an entry has an empty camera identifier",
				ErrMalformedFile, name)
		}
		if fe.User == "" {
			// networking.CallMethod guards the UsernameToken on the username alone
			// (networking/client.go:239-253, pinned by networking/wssecurity_test.go), so
			// an entry without one authenticates nothing and is almost always a "usr" or
			// "passwd" typo. An empty password, by the same test, is legitimate.
			return fmt.Errorf("%w: %s: entry %q has no user", ErrMalformedFile, name, id)
		}
		if previous, seen := s.byID[key]; seen {
			return fmt.Errorf("%w: %q, in %s and %s", ErrDuplicateEntry, id, previous.file, name)
		}
		s.byID[key] = entry{
			auth: networking.ClientAuth{Username: fe.User, Password: fe.Password},
			file: name,
		}
	}
	return nil
}

// jsonFault renders a decoder failure without quoting the document.
//
// json.SyntaxError's message embeds the offending character, and that character is a
// character of the file -- possibly of the password. So the decoder's error is never
// wrapped with %w and never printed: only positions and type names cross this boundary.
//
// The UnmarshalTypeError branch is safe only because every field of fileEntry is a string.
// encoding/json sets Value to "number <literal>" when a numeric literal fails to fit a
// numeric target, so adding an int field, or a json:",string" tag, would start printing an
// unquoted password. TestNoErrorMentionsAPassword covers that case so the change fails
// loudly rather than silently.
func jsonFault(err error) string {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return fmt.Sprintf("invalid JSON at byte %d", syntax.Offset)
	}
	var typ *json.UnmarshalTypeError
	if errors.As(err, &typ) {
		if typ.Field != "" {
			return fmt.Sprintf("field %q is a %s, want %s (byte %d)",
				typ.Field, typ.Value, typ.Type, typ.Offset)
		}
		return fmt.Sprintf("a %s was found where a %s was expected (byte %d)",
			typ.Value, typ.Type, typ.Offset)
	}
	return "the file is not an object mapping a camera identifier to a user and a password"
}

// Resolve implements Resolver. The source it reports is the file the entry came from.
func (s *Store) Resolve(id string) (networking.ClientAuth, string, bool) {
	if s == nil {
		return networking.ClientAuth{}, "", false
	}
	e, ok := s.byID[CanonicalID(id)]
	if !ok {
		return networking.ClientAuth{}, "", false
	}
	return e.auth, e.file, true
}

// Len is the number of cameras the Store knows about.
func (s *Store) Len() int {
	if s == nil {
		return 0
	}
	return len(s.byID)
}

// LaxPermissions returns, in directory order, the files whose mode grants any access to
// group or other. It is a signal, not a verdict: they have been loaded regardless, because
// a workstation where every file is 0664 must keep working.
func (s *Store) LaxPermissions() []string {
	if s == nil {
		return nil
	}
	return s.lax
}

// String reports a count and never an entry. fmt prints unexported fields, so the default
// rendering of this struct would print every password it holds.
func (s *Store) String() string {
	return fmt.Sprintf("credentials.Store(%d entries)", s.Len())
}

// GoString is String. %#v ignores String entirely and prints every unexported field, which
// is exactly the verb someone reaches for when debugging a struct like this one, so
// without this the Go-syntax rendering prints the whole entry map, passwords included.
func (s *Store) GoString() string { return s.String() }

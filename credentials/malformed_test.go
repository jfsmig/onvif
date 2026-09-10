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

// A credentials file is hand-written, so it is mistyped. This package may not log
// (AGENTS.md: packages below sdk return errors and log nothing), so an error is the only
// channel by which an operator learns that the file they wrote does nothing -- and a
// loader that shrugged a bad file off would reproduce exactly the unexplained 401 the
// per-camera files exist to remove.
//
// Each case below is one way a file can be wrong, and each is a failure rather than a
// silent skip on purpose.

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadRejectsMalformedJSON(t *testing.T) {
	_, err := LoadFS(fstest.MapFS{"a.json": file(`{"` + camA + `": {"user": "admin",`)})
	if !errors.Is(err, ErrMalformedFile) {
		t.Fatalf("err = %v, want it to match ErrMalformedFile", err)
	}
	if !strings.Contains(err.Error(), "a.json") {
		t.Errorf("err = %v, want the file named so the operator knows which one to open", err)
	}
}

func TestLoadRejectsAnEmptyFile(t *testing.T) {
	// A zero-byte file is a truncated write, not an intention.
	for _, name := range []string{"empty", "whitespace"} {
		content := ""
		if name == "whitespace" {
			content = "\n  \n"
		}
		t.Run(name, func(t *testing.T) {
			if _, err := LoadFS(fstest.MapFS{"a.json": file(content)}); !errors.Is(err, ErrMalformedFile) {
				t.Fatalf("err = %v, want it to match ErrMalformedFile", err)
			}
		})
	}
}

func TestLoadAcceptsAnEmptyObject(t *testing.T) {
	// "{}" is how a file says it deliberately holds nothing, and it must be distinguishable
	// from the truncated write above.
	store, err := LoadFS(fstest.MapFS{"a.json": file(`{}`)})
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if store.Len() != 0 {
		t.Errorf("Len() = %d, want 0", store.Len())
	}
}

func TestLoadRejectsDuplicateIdentifiers(t *testing.T) {
	// Two entries for one camera may disagree, and settling that by the order fs.ReadDir
	// happens to return would make the effective password a function of the file names.
	t.Run("across two files", func(t *testing.T) {
		_, err := LoadFS(fstest.MapFS{
			"a.json": file(jsonFor(camA, "admin", "one")),
			"b.json": file(jsonFor(camA, "admin", "two")),
		})
		if !errors.Is(err, ErrDuplicateEntry) {
			t.Fatalf("err = %v, want it to match ErrDuplicateEntry", err)
		}
		for _, name := range []string{"a.json", "b.json"} {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("err = %v, want both files named", err)
			}
		}
	})

	t.Run("two spellings in one file", func(t *testing.T) {
		// The two forms ONVIF Core 7.1 and WS-Discovery 2.6 disagree about are the same
		// camera once canonicalised, so this collision is real and detectable.
		_, err := LoadFS(fstest.MapFS{"a.json": file(`{
			"urn:uuid:` + bareA + `": {"user": "admin", "password": "one"},
			"uuid:` + bareA + `": {"user": "admin", "password": "two"}
		}`)})
		if !errors.Is(err, ErrDuplicateEntry) {
			t.Fatalf("err = %v, want it to match ErrDuplicateEntry", err)
		}
	})
}

func TestLoadRejectsAnEmptyIdentifier(t *testing.T) {
	// An empty key would sit in the map matching nothing while looking like it matches
	// everything.
	for _, key := range []string{"", "   ", "urn:uuid:"} {
		if _, err := LoadFS(fstest.MapFS{"a.json": file(jsonFor(key, "admin", "secret"))}); !errors.Is(err, ErrMalformedFile) {
			t.Errorf("key %q: err = %v, want it to match ErrMalformedFile", key, err)
		}
	}
}

func TestLoadRejectsAnEntryWithoutAUser(t *testing.T) {
	// CallMethod guards the UsernameToken on the username alone (networking/client.go:239-253),
	// so an entry with no user authenticates nothing at all. In practice it is a "usr" or
	// "passwd" typo, and failing at load beats a 401 nobody can explain.
	_, err := LoadFS(fstest.MapFS{"a.json": file(`{"` + camA + `": {"usr": "admin", "passwd": "secret"}}`)})
	if !errors.Is(err, ErrMalformedFile) {
		t.Fatalf("err = %v, want it to match ErrMalformedFile", err)
	}
	if !strings.Contains(err.Error(), camA) {
		t.Errorf("err = %v, want the entry named", err)
	}
}

func TestLoadAcceptsAnEmptyPassword(t *testing.T) {
	// An ONVIF account may legitimately have an empty password, and
	// networking/wssecurity_test.go pins that such a client still sends a UsernameToken.
	// The entry is therefore valid and must not be swept up by the rule above.
	store, err := LoadFS(fstest.MapFS{"a.json": file(jsonFor(camA, "admin", ""))})
	if err != nil {
		t.Fatalf("LoadFS refused an empty password: %v", err)
	}
	if auth, _, ok := store.Resolve(camA); !ok || auth.Username != "admin" || auth.Password != "" {
		t.Errorf("Resolve = (%q/%q, %v), want the entry with its empty password", auth.Username, auth.Password, ok)
	}
}

// unreadableFS is a directory whose one file cannot be read. fstest.MapFS cannot express
// that -- its Open ignores the mode -- and a real 0000 file would still be readable by the
// root account CI may well run as.
//
// Both methods are overridden because both are reachable: fs.ReadFile takes the ReadFileFS
// shortcut when the filesystem offers one, and os.DirFS -- what Load really uses -- does.
type unreadableFS struct{ fstest.MapFS }

func (u unreadableFS) Open(name string) (fs.File, error) {
	if strings.HasSuffix(name, ".json") {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return u.MapFS.Open(name)
}

func (u unreadableFS) ReadFile(name string) ([]byte, error) {
	if strings.HasSuffix(name, ".json") {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return u.MapFS.ReadFile(name)
}

func TestLoadReportsAnUnreadableFile(t *testing.T) {
	// A file that cannot be read is the reason the camera will 401. Failing once at
	// startup beats running the whole fleet unauthenticated.
	_, err := LoadFS(unreadableFS{fstest.MapFS{"a.json": file(jsonFor(camA, "admin", "secret"))}})
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("err = %v, want it to match fs.ErrPermission", err)
	}
}

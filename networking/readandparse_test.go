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

package networking

// Two slips in ReadAndParse, both invisible from the outside:
//
//   - the tag naming the ONVIF operation was accepted and discarded, so a parse failure
//     read "expected element type <Envelope>" with nothing to say which of the SDK's ~75
//     calls produced it. sdk.Fetch* returns no error either, so that message was all an
//     operator got. All 205 generated wrappers were already passing the name.
//   - io.ReadAll had no bound, so a device streaming without end could exhaust the
//     caller's memory. SOAP 1.2 Part 1 section 5 makes an envelope one finite XML
//     document, so an unbounded reply is by definition not a conformant response.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jfsmig/onvif/utils"
)

// reply performs one exchange against a handler and hands back the live response, so the
// body is still unread when ReadAndParse gets it.
func reply(t *testing.T, handler http.HandlerFunc) *http.Response {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	resp, err := SendSoap(context.Background(), srv.Client(), srv.URL, "<Probe/>")
	if err != nil {
		t.Fatalf("SendSoap: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// TestReadAndParseNamesTheOperationInItsError covers both error paths, because the tag was
// dropped on both.
func TestReadAndParseNamesTheOperationInItsError(t *testing.T) {
	t.Run("malformed body", func(t *testing.T) {
		resp := reply(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("this is not XML at all"))
		})

		err := ReadAndParse(resp, &struct{}{}, "GetCapabilities")
		if err == nil {
			t.Fatal("ReadAndParse accepted a body that is not XML")
		}
		if !strings.Contains(err.Error(), "GetCapabilities") {
			t.Errorf("error %q does not name the operation, so it cannot be traced to a call", err)
		}
	})

	t.Run("non-200", func(t *testing.T) {
		resp := reply(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		err := ReadAndParse(resp, &struct{}{}, "GetProfiles")
		if err == nil {
			t.Fatal("ReadAndParse accepted a 500")
		}
		// ErrHTTP must stay the first wrapped error: callers match on it, and
		// redirect_test.go depends on that too.
		if !errors.Is(err, utils.ErrHTTP) {
			t.Errorf("error %v no longer matches utils.ErrHTTP", err)
		}
		if !strings.Contains(err.Error(), "GetProfiles") {
			t.Errorf("error %q does not name the operation", err)
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error %q dropped the status, which is what tells 401 from 500", err)
		}
	})
}

// TestReadAndParseBoundsTheResponseSize points ReadAndParse at a device that answers
// forever. Without the bound this buffers until the process dies; with it, the read stops
// at the cap and the error says so.
func TestReadAndParseBoundsTheResponseSize(t *testing.T) {
	// Served in chunks rather than one huge allocation, so the test does not itself need
	// MaxResponseBytes of memory to prove the point.
	chunk := strings.Repeat("A", 1<<16)

	resp := reply(t, func(w http.ResponseWriter, r *http.Request) {
		for {
			if _, err := w.Write([]byte(chunk)); err != nil {
				return // ReadAndParse stopped reading; that is the pass condition
			}
		}
	})

	done := make(chan error, 1)
	go func() { done <- ReadAndParse(resp, &struct{}{}, "GetSystemLog") }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("ReadAndParse accepted an endless reply")
		}
		if !strings.Contains(err.Error(), "GetSystemLog") {
			t.Errorf("error %q does not name the operation", err)
		}
		if !strings.Contains(err.Error(), "exceeds") {
			t.Errorf("error %q does not report the size limit, so the failure looks like "+
				"malformed XML rather than an oversized reply", err)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("ReadAndParse did not stop reading; the response size is unbounded")
	}
}

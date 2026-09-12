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

// The SOAP fault used to be thrown away whole.
//
// ONVIF Core section 5.11.2.1 makes the SOAP 1.2 fault the only channel for an operation
// error, and section 5.11.2.2 Table 5 makes the ter: Subcode the normative discriminator:
// ter:NotAuthorized, ter:InvalidArgVal, ter:ActionNotSupported, ter:WellFormed,
// ter:TagMismatch. Under the SOAP 1.2 HTTP binding an env:Sender fault arrives as 400 and an
// env:Receiver fault as 500, so ReadAndParse's status-only report collapsed "wrong password",
// "unsupported operation" and "malformed request" into one indistinguishable
// "http request error: GetProfiles: 400 Bad Request".
//
// The 200 case is the worse half: the generated Envelope has no field a soap:Fault can match,
// so xml.Unmarshal succeeded and the call returned a zero response and a nil error. Firmware
// that answers a fault with 200 is not conformant and is common, and sdk.Fetch* cannot tell
// that apart from a camera with genuinely nothing to say.
const faultNotAuthorized = `<?xml version="1.0"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope"
              xmlns:ter="http://www.onvif.org/ver10/error">
 <env:Body><env:Fault>
  <env:Code><env:Value>env:Sender</env:Value>
   <env:Subcode><env:Value>ter:NotAuthorized</env:Value></env:Subcode></env:Code>
  <env:Reason><env:Text xml:lang="en">Sender not Authorized</env:Text></env:Reason>
 </env:Fault></env:Body>
</env:Envelope>`

func TestReadAndParseSurfacesTheSOAPFault(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError} {
		resp := reply(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/soap+xml")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(faultNotAuthorized))
		})

		var out struct {
			Body struct{ GetProfilesResponse struct{} }
		}
		err := ReadAndParse(resp, &out, "GetProfiles")
		if err == nil {
			t.Fatalf("status %d: a SOAP Fault parsed as a successful reply", status)
		}
		if !errors.Is(err, utils.ErrSOAPFault) {
			t.Errorf("status %d: error %v does not match utils.ErrSOAPFault", status, err)
		}
		// The subcode is the whole point: it is what tells a rejected credential from an
		// operation the device does not implement.
		if !strings.Contains(err.Error(), "ter:NotAuthorized") {
			t.Errorf("status %d: error %q does not name the fault subcode", status, err)
		}
		if !strings.Contains(err.Error(), "GetProfiles") {
			t.Errorf("status %d: error %q does not name the operation", status, err)
		}
		// A faulted non-200 must keep matching ErrHTTP as well, because callers already
		// match on it and a fault is additional information, not a different failure.
		if status != http.StatusOK && !errors.Is(err, utils.ErrHTTP) {
			t.Errorf("status %d: error %v no longer matches utils.ErrHTTP", status, err)
		}
	}
}

// An authentication failure has to be distinguishable from every other fault.
//
// ONVIF makes whole services and many operations conditional, so most faults mean "this
// camera does not do that" -- an answer about the camera, which sdk swallows into an empty
// field on purpose. A rejected credential is not that: every other call will fail the same
// way, and the empty result the operator is left with says nothing about why. Against the
// bench, one `dump all` on a camera with the wrong password produced 41 identical faults and
// not one word above trace level.
//
// Two shapes reach us, and both must match. A conformant device sends the SOAP fault with
// subcode ter:NotAuthorized (Core section 5.11.2.2, Table 5); a device that rejects the
// credentials at the HTTP layer sends a bare 401 or 403 with nothing to parse.
func TestReadAndParseRecognisesARejectedCredential(t *testing.T) {
	const fault = `<?xml version="1.0"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope"
              xmlns:ter="http://www.onvif.org/ver10/error">
 <env:Body><env:Fault>
  <env:Code><env:Value>env:Sender</env:Value>
   <env:Subcode><env:Value>ter:NotAuthorized</env:Value></env:Subcode></env:Code>
  <env:Reason><env:Text xml:lang="en">Sender not Authorized</env:Text></env:Reason>
 </env:Fault></env:Body>
</env:Envelope>`

	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"fault subcode", http.StatusBadRequest, fault, true},
		{"fault subcode on a 200", http.StatusOK, fault, true},
		{"bare 401", http.StatusUnauthorized, "not xml at all", true},
		{"bare 403", http.StatusForbidden, "", true},
		// The discriminator has to discriminate: an operation the device does not
		// implement is the ordinary case, and must stay ordinary.
		{"unsupported operation", http.StatusBadRequest,
			strings.Replace(fault, "ter:NotAuthorized", "ter:ActionNotSupported", 1), false},
		{"server error", http.StatusInternalServerError, "boom", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := reply(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/soap+xml")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})

			var out struct {
				Body struct{ GetProfilesResponse struct{} }
			}
			err := ReadAndParse(resp, &out, "GetProfiles")
			if err == nil {
				t.Fatal("no error at all")
			}
			if got := errors.Is(err, utils.ErrNotAuthorized); got != tc.want {
				t.Errorf("errors.Is(%v, ErrNotAuthorized) = %v, want %v", err, got, tc.want)
			}
		})
	}
}

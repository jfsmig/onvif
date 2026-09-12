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

// A dump that ran out of time printed a complete-looking camera and exited 0.
//
// sdk.Fetch* swallows a per-call failure on purpose, and that is right: cameras vary wildly
// in what they implement, and one unsupported operation must not lose the whole dump. A
// context expiry is the other thing entirely. It is our own deadline -- runOneShot puts a
// minute around every command -- and it says nothing about the camera at all: every field is
// zero because nothing was asked, not because nothing was there. Encoding that produced a
// syntactically perfect document full of zeroes, with nothing on stderr below -vvv.
//
// ErrNoProfileS, twenty lines above the defect, already names this failure mode in the
// author's own words: "emitting an object full of zero values would look like a camera that
// answered with nothing rather than one that was never asked." It was guarded for the
// missing-Profile-S path and not for the deadline. subscribe.go checks ctx.Err() at six
// separate points; the dump checked it nowhere.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
)

func dumpSoap(body string) string {
	return `<?xml version="1.0"?><s:Envelope ` +
		`xmlns:s="http://www.w3.org/2003/05/soap-envelope" ` +
		`xmlns:tds="http://www.onvif.org/ver10/device/wsdl" ` +
		`xmlns:trt="http://www.onvif.org/ver10/media/wsdl" ` +
		`xmlns:tt="http://www.onvif.org/ver10/schema"><s:Body>` + body + `</s:Body></s:Envelope>`
}

// slowCamera answers the two bootstrap exchanges NewDevice needs, and then blocks every
// later request until its context is done -- a camera on a slow link, or one behind a
// MaxConnsPerHost queue forty operations deep, both of which end the same way.
func slowCamera(t *testing.T) *httptest.Server {
	t.Helper()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = io.ReadFull(r.Body, buf)
		req := string(buf)
		host := strings.TrimPrefix(srv.URL, "http://")

		var out string
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			out = dumpSoap(`<tds:GetSystemDateAndTimeResponse/>`)
		case strings.Contains(req, "GetCapabilities"):
			out = dumpSoap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
				`<tt:Device><tt:XAddr>http://` + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
				`<tt:Media><tt:XAddr>http://` + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
				`</tds:Capabilities></tds:GetCapabilitiesResponse>`)
		default:
			<-r.Context().Done()
			return
		}
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(out))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// captureStdout runs fn with os.Stdout replaced, and returns what it wrote. dumpSomething
// encodes straight to os.Stdout, which is the property under test: nothing at all must
// arrive there when the dump is known to be incomplete.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w

	var buf bytes.Buffer
	var drained sync.WaitGroup
	drained.Go(func() { _, _ = io.Copy(&buf, r) })

	fn()

	os.Stdout = saved
	_ = w.Close()
	drained.Wait()
	_ = r.Close()
	return buf.String()
}

func TestADumpThatRanOutOfTimeIsNotPrinted(t *testing.T) {
	srv := slowCamera(t)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var err error
	stdout := captureStdout(t, func() {
		err = dumpSomething(ctx,
			networking.ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")},
			func(app sdk.Appliance, s *sdk.ProfileS) interface{} { return s.FetchMediaProfiles(ctx) })
	})

	if err == nil {
		t.Fatal("a dump whose deadline expired reported success, so the tool exits 0 " +
			"on a document that is zeroes all the way down")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error %v does not report the expired deadline", err)
	}
	if stdout != "" {
		t.Errorf("an incomplete dump reached stdout, where it is indistinguishable "+
			"from a camera that answered nothing: %q", stdout)
	}
}

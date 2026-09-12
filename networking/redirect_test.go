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

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jfsmig/onvif/v2/utils"
)

// The ONVIF credential travels inside the SOAP body, so net/http's own protection — it
// strips Authorization and Cookie across hosts — does not cover it. A 307 or 308 replays
// the body verbatim, which handed the WS-Security UsernameToken to whatever host the
// Location named. These tests pin that shut.

const secretBody = `<Envelope><Header><UsernameToken>SUPER-SECRET-DIGEST</UsernameToken></Header></Envelope>`

// redirectPair builds a "camera" that answers 307 pointing at an "attacker", and reports
// how much each of them received.
func redirectPair(t *testing.T) (cameraURL string, cameraHits, attackerHits *atomic.Int32) {
	t.Helper()

	cameraHits, attackerHits = &atomic.Int32{}, &atomic.Int32{}

	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attackerHits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(attacker.Close)

	camera := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cameraHits.Add(1)
		http.Redirect(w, r, attacker.URL+"/collect", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(camera.Close)

	return camera.URL, cameraHits, attackerHits
}

func TestRedirectDoesNotReplayCredentials(t *testing.T) {
	for _, tc := range []struct {
		name     string
		supplied *http.Client
	}{
		{"nil client, NewClient supplies one", nil},
		{"caller-supplied client with no CheckRedirect", &http.Client{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cameraURL, cameraHits, attackerHits := redirectPair(t)

			client, err := NewClient(ClientInfo{Xaddr: strings.TrimPrefix(cameraURL, "http://")}, tc.supplied)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			resp, err := SendSoap(context.Background(), client.httpClient, cameraURL, secretBody)
			if err != nil {
				t.Fatalf("SendSoap: %v", err)
			}
			defer resp.Body.Close()

			// The security assertion.
			if n := attackerHits.Load(); n != 0 {
				t.Fatalf("the redirect target received %d request(s); the SOAP body carrying the "+
					"WS-Security token was replayed off-host", n)
			}
			// Guard against a vacuous pass: the request must really have been sent.
			if n := cameraHits.Load(); n != 1 {
				t.Fatalf("camera received %d requests, want 1 — the test is not exercising the request path", n)
			}
			// The 3xx comes back to the caller so the failure stays diagnosable.
			if resp.StatusCode != http.StatusTemporaryRedirect {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusTemporaryRedirect)
			}
			if err := ReadAndParse(resp, &struct{}{}, "Probe"); err == nil {
				t.Fatal("ReadAndParse accepted a 3xx")
			} else if !errors.Is(err, utils.ErrHTTP) {
				t.Fatalf("error %v does not match utils.ErrHTTP", err)
			} else if !strings.Contains(err.Error(), "307") {
				t.Fatalf("error %q does not carry the status", err)
			}
		})
	}
}

// A caller who has made their own redirect decision keeps it.
func TestCallerCheckRedirectIsPreserved(t *testing.T) {
	cameraURL, _, _ := redirectPair(t)

	var called atomic.Int32
	supplied := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			called.Add(1)
			return http.ErrUseLastResponse
		},
	}

	client, err := NewClient(ClientInfo{Xaddr: strings.TrimPrefix(cameraURL, "http://")}, supplied)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := SendSoap(context.Background(), client.httpClient, cameraURL, secretBody)
	if err != nil {
		t.Fatalf("SendSoap: %v", err)
	}
	defer resp.Body.Close()

	if called.Load() != 1 {
		t.Fatalf("caller's CheckRedirect invoked %d times, want 1 — it was overridden", called.Load())
	}
}

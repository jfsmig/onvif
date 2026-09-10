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

// CallMethod used to require a non-empty username *and* a non-empty password before it
// would add the wsse:Security header. An ONVIF account may legitimately have an empty
// password, and such a client got no UsernameToken at all rather than a token carrying an
// empty one -- so it fell through to whatever the device does for an unauthenticated
// request, which is a 401 on anything past the pre-auth operations.
//
// The guard is now on the username alone, so an entirely empty ClientAuth -- which is what
// the sdk stub tests use -- still sends nothing.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// probeRequest is marshalled by CallMethod like any request struct. Being declared in this
// package, its PkgPath ends in "networking", which is the endpoint key the tests below
// register -- CallMethod routes on that last segment.
type probeRequest struct {
	XMLName string `xml:"tds:GetSystemDateAndTime"`
}

// captureEnvelope performs one CallMethod against a stub and returns the envelope that
// reached the wire.
func captureEnvelope(t *testing.T, auth ClientAuth) string {
	t.Helper()

	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		body = string(buf)
		_, _ = w.Write([]byte(`<Envelope/>`))
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient(ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")}, srv.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.SetAuth(auth)
	client.AddEndpoint("networking", srv.URL)

	resp, err := client.CallMethod(context.Background(), probeRequest{})
	if err != nil {
		t.Fatalf("CallMethod: %v", err)
	}
	_ = resp.Body.Close()

	return body
}

func TestEmptyPasswordStillSendsAUsernameToken(t *testing.T) {
	t.Run("username with an empty password", func(t *testing.T) {
		env := captureEnvelope(t, ClientAuth{Username: "admin", Password: ""})

		if !strings.Contains(env, "UsernameToken") {
			t.Fatal("no UsernameToken sent for an account whose password is empty; " +
				"the request goes out unauthenticated")
		}
		if !strings.Contains(env, "admin") {
			t.Error("the token does not carry the username")
		}
		// Nonce and Created are what a device validates the digest against, so their
		// presence is what makes the token usable at all.
		for _, want := range []string{"Nonce", "Created", "Password"} {
			if !strings.Contains(env, want) {
				t.Errorf("the token is missing %s", want)
			}
		}
	})

	t.Run("no credentials at all", func(t *testing.T) {
		env := captureEnvelope(t, ClientAuth{})

		if strings.Contains(env, "UsernameToken") {
			t.Error("a UsernameToken was sent for an empty ClientAuth; callers that pass " +
				"no credentials must stay unauthenticated")
		}
	})

	t.Run("password with an empty username", func(t *testing.T) {
		// Not a real configuration, but it is the other half of the old && guard: a
		// password without a username identifies nobody, so nothing should be sent.
		env := captureEnvelope(t, ClientAuth{Password: "secret"})

		if strings.Contains(env, "UsernameToken") {
			t.Error("a UsernameToken was sent with no username")
		}
		if strings.Contains(env, "secret") {
			t.Error("the password reached the wire with no username to go with it")
		}
	})
}

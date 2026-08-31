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
	"testing"
	"time"
)

// req.WithContext returns a copy; discarding it left every request on context.Background(),
// so a device that accepted the connection and then went silent blocked forever — past the
// caller's deadline, and past SIGINT. Neither the compiler nor go vet flags that, so these
// tests are the only thing standing between the repo and a silent regression.

// blackHole is a server that accepts the request and never responds, until the test ends.
func blackHole(t *testing.T) string {
	t.Helper()
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	t.Cleanup(func() { close(release); srv.Close() })
	return srv.URL
}

func TestContextDeadlineBoundsTheRequest(t *testing.T) {
	url := blackHole(t)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := SendSoap(ctx, &http.Client{}, url, "<Envelope/>")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("SendSoap returned no error against a server that never responds")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error %v is not a context deadline; the context is not reaching the request", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("took %v to honour a 250ms deadline", elapsed)
	}
}

func TestContextCancelUnblocksTheRequest(t *testing.T) {
	url := blackHole(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel() // stands in for SIGINT
	}()

	start := time.Now()
	_, err := SendSoap(ctx, &http.Client{}, url, "<Envelope/>")
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v is not context.Canceled; cancellation is not reaching the request", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("took %v to honour a cancel at 200ms", elapsed)
	}
}

// A caller who supplies no deadline still gets a bound, from the client NewClient builds.
func TestDefaultClientCarriesATimeout(t *testing.T) {
	client, err := NewClient(ClientInfo{Xaddr: "192.0.2.1:80"}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.httpClient.Timeout != DefaultTimeout {
		t.Fatalf("default client Timeout = %v, want %v", client.httpClient.Timeout, DefaultTimeout)
	}
}

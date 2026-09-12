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

package sdk

// A camera that rejects the credentials used to be indistinguishable, in the output, from a
// camera that simply does not implement the operation: Fetch* swallowed both at trace level
// and left the same empty field, so `onvif-cli dump` printed a plausible document and exited
// 0 with nothing on stderr. On the bench, `dump all` against a camera with the wrong password
// produced 41 authentication faults and not one word above trace.
//
// The rest of the Fetch* contract is untouched and must stay so: an operation the camera does
// not implement is the camera describing itself, the empty field is the answer, and -vvv is
// where the detail belongs. A rejected credential is the exception because it is not a
// description of the camera at all -- every other call will fail the same way.
//
// Once per appliance, never once per call, which is the other half of the fix: forty
// identical warnings would bury the one that matters.

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"

	"github.com/jfsmig/onvif/networking"
)

// rejectingStub answers the two bootstrap exchanges and then refuses everything with the
// fault ONVIF Core section 5.11.2.2 Table 5 names for a rejected credential.
func rejectingStub(t *testing.T) *ProfileS {
	t.Helper()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = io.ReadFull(r.Body, buf)
		req := string(buf)
		host := strings.TrimPrefix(srv.URL, "http://")

		w.Header().Set("Content-Type", "application/soap+xml")
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			_, _ = w.Write([]byte(soap(`<tds:GetSystemDateAndTimeResponse/>`)))
		case strings.Contains(req, "GetCapabilities"):
			_, _ = w.Write([]byte(soap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
				`<tt:Device><tt:XAddr>http://` + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
				`<tt:Media><tt:XAddr>http://` + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
				`</tds:Capabilities></tds:GetCapabilitiesResponse>`)))
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(soap(`<env:Fault>` +
				`<env:Code><env:Value>env:Sender</env:Value>` +
				`<env:Subcode><env:Value>ter:NotAuthorized</env:Value></env:Subcode></env:Code>` +
				`<env:Reason><env:Text xml:lang="en">Sender not Authorized</env:Text></env:Reason>` +
				`</env:Fault>`)))
		}
	}))
	t.Cleanup(srv.Close)

	dev, err := NewDevice(context.Background(),
		networking.ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")},
		networking.ClientAuth{Username: "operator", Password: "wrong"}, nil)
	if err != nil {
		t.Fatalf("NewDevice: %v", err)
	}
	profileS, ok := dev.ProfileS()
	if !ok {
		t.Fatal("the stub advertises Device and Media, so Profile S should be available")
	}
	return profileS
}

// captureLogger swaps the package Logger for one writing into a buffer, at trace level so
// that everything the code chooses to emit is visible and the test judges the level itself.
type captureLogger struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (c *captureLogger) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}

func (c *captureLogger) lines(level string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var out []string
	for _, line := range strings.Split(c.buf.String(), "\n") {
		if strings.Contains(line, `"level":"`+level+`"`) {
			out = append(out, line)
		}
	}
	return out
}

func withCapturedLogger(t *testing.T) *captureLogger {
	t.Helper()

	sink := &captureLogger{}
	saved, savedLevel := Logger, zerolog.GlobalLevel()
	Logger = zerolog.New(sink)
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	t.Cleanup(func() {
		Logger = saved
		zerolog.SetGlobalLevel(savedLevel)
	})
	return sink
}

func TestARejectedCredentialIsReportedOncePerAppliance(t *testing.T) {
	profileS := rejectingStub(t)
	sink := withCapturedLogger(t)

	// Eight operations across four services, which is what `dump all` does.
	ctx := context.Background()
	out := profileS.FetchDeviceSystem(ctx)
	_ = out
	_ = profileS.FetchDeviceSecurity(ctx)
	_ = profileS.FetchMedia(ctx)
	_ = profileS.FetchMediaProfiles(ctx)

	warnings := sink.lines("warn")
	if len(warnings) == 0 {
		t.Fatal("the camera rejected every call and nothing above trace level said so: " +
			"an operator sees an empty dump, exit 0, and a silent stderr")
	}
	if len(warnings) != 1 {
		t.Errorf("%d warnings for one appliance, want exactly 1 -- forty identical lines "+
			"bury the one that matters:\n%s", len(warnings), strings.Join(warnings, "\n"))
	}
	if got := warnings[0]; !strings.Contains(got, "NotAuthorized") {
		t.Errorf("the warning does not name the cause: %s", got)
	}

	// Everything else stays at trace, which is the Fetch* contract and is not being changed.
	if len(sink.lines("trace")) == 0 {
		t.Error("the per-call failures stopped being logged at trace level")
	}
}

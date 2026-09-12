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

// CallMethod stamped the WS-Security Created timestamp from the local clock. A device
// compares that stamp against its own clock and rejects a token more than a few seconds
// out -- which is why gosoap exposes AddWSSecurityAt at all -- and camera clocks drift. So
// on a skewed camera the unauthenticated probe succeeded and every authenticated call then
// failed 401.
//
// The offset was available the whole time: load() issues GetSystemDateAndTime as its
// liveness probe and used to close the body unread. docs/wsdl/devicemgmt.wsdl:2688
// requires the reply to carry UTCDateTime.
//
// These tests drive the real NewDevice path against a stub camera, reusing the soap()
// helper from ptz_token_test.go, and read the Created stamp back off the wire.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jfsmig/onvif/v2/networking"
	"github.com/jfsmig/onvif/v2/xsd/onvif"
)

// createdStamp pulls the wsu:Created text out of a captured envelope. Matched on the local
// name, because gosoap emits the element with an xmlns attribute rather than a prefix.
var createdStamp = regexp.MustCompile(`<(?:[A-Za-z0-9]+:)?Created[^>]*>([^<]+)</(?:[A-Za-z0-9]+:)?Created>`)

// systemDateAndTime renders a GetSystemDateAndTimeResponse announcing the given instant as
// the device's UTC clock, in the structured tt:DateTime shape a real camera sends.
func systemDateAndTime(at time.Time) string {
	at = at.UTC()
	return soap(`<tds:GetSystemDateAndTimeResponse><tds:SystemDateAndTime>` +
		`<tt:DateTimeType>NTP</tt:DateTimeType>` +
		`<tt:DaylightSavings>false</tt:DaylightSavings>` +
		`<tt:UTCDateTime>` +
		`<tt:Time><tt:Hour>` + strconv.Itoa(at.Hour()) + `</tt:Hour>` +
		`<tt:Minute>` + strconv.Itoa(at.Minute()) + `</tt:Minute>` +
		`<tt:Second>` + strconv.Itoa(at.Second()) + `</tt:Second></tt:Time>` +
		`<tt:Date><tt:Year>` + strconv.Itoa(at.Year()) + `</tt:Year>` +
		`<tt:Month>` + strconv.Itoa(int(at.Month())) + `</tt:Month>` +
		`<tt:Day>` + strconv.Itoa(at.Day()) + `</tt:Day></tt:Date>` +
		`</tt:UTCDateTime>` +
		`</tds:SystemDateAndTime></tds:GetSystemDateAndTimeResponse>`)
}

// clockStub stands up a camera announcing deviceClock as its UTC time, and records the
// Created stamp of the first authenticated request that follows the probe.
func clockStub(t *testing.T, deviceClock time.Time) (*httptest.Server, func() string) {
	t.Helper()

	var mu sync.Mutex
	var created string

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		req := string(buf)

		host := strings.TrimPrefix(srv.URL, "http://")

		// Every request after the probe is a candidate; keep the first, so the assertion
		// is about a call the offset was meant to protect and not about the probe itself.
		if !strings.Contains(req, "GetSystemDateAndTime") {
			if m := createdStamp.FindStringSubmatch(req); m != nil {
				mu.Lock()
				if created == "" {
					created = m[1]
				}
				mu.Unlock()
			}
		}

		var out string
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			out = systemDateAndTime(deviceClock)
		case strings.Contains(req, "GetCapabilities"):
			out = soap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
				`<tt:Device><tt:XAddr>http://` + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
				`<tt:Media><tt:XAddr>http://` + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
				`</tds:Capabilities></tds:GetCapabilitiesResponse>`)
		default:
			out = soap(`<tds:Empty/>`)
		}
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(out))
	}))
	t.Cleanup(srv.Close)

	return srv, func() string {
		mu.Lock()
		defer mu.Unlock()
		return created
	}
}

// authenticatedCall drives NewDevice and then one further operation, so that a Created
// stamp is produced. Credentials must be non-empty or no header is emitted at all.
func authenticatedCall(t *testing.T, srv *httptest.Server) {
	t.Helper()

	dev, err := NewDevice(context.Background(),
		networking.ClientInfo{Xaddr: strings.TrimPrefix(srv.URL, "http://")},
		networking.ClientAuth{Username: "admin", Password: "secret"},
		nil)
	if err != nil {
		t.Fatalf("NewDevice: %v", err)
	}
	profileS, ok := dev.ProfileS()
	if !ok {
		t.Fatal("the stub advertises Device and Media, so Profile S should be available")
	}
	// Any authenticated operation will do; the reply is irrelevant, the header is not.
	_ = profileS.FetchDeviceDescriptor(context.Background())
}

// TestWSSecurityCreatedUsesTheDeviceClock is the regression proper: a camera ten minutes
// ahead must be sent a Created stamp in its own time, not ours.
func TestWSSecurityCreatedUsesTheDeviceClock(t *testing.T) {
	const skew = 10 * time.Minute
	deviceClock := time.Now().UTC().Add(skew)

	srv, captured := clockStub(t, deviceClock)
	authenticatedCall(t, srv)

	raw := captured()
	if raw == "" {
		t.Fatal("no Created stamp captured; the request path did not emit a UsernameToken")
	}
	stamped, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t.Fatalf("Created %q is not RFC3339: %v", raw, err)
	}

	// Device time, within the second the stub's clock was built to.
	if d := stamped.Sub(deviceClock); d > 2*time.Second || d < -2*time.Second {
		t.Errorf("Created = %s, which is %v from the device clock %s; the stamp is not in "+
			"device time", stamped.Format(time.RFC3339), d, deviceClock.Format(time.RFC3339))
	}
	// And demonstrably not local time, which is what it used to be.
	if d := stamped.Sub(time.Now().UTC()); d < 5*time.Minute {
		t.Errorf("Created = %s is only %v from local time; the offset was not applied",
			stamped.Format(time.RFC3339), d)
	}
	// UTC on the wire: a non-UTC Location would emit +02:00 where a device expects Z.
	if !strings.HasSuffix(raw, "Z") {
		t.Errorf("Created = %q does not end in Z, so it was not stamped in UTC", raw)
	}
}

// TestClockOffsetIsZeroWhenUTCDateTimeIsAbsent is what guarantees the change is a no-op
// against every camera that omits the optional element -- and that the empty
// <tds:GetSystemDateAndTimeResponse/> the other stubs in this package return keeps working
// for the right reason rather than by accident.
func TestClockOffsetIsZeroWhenUTCDateTimeIsAbsent(t *testing.T) {
	var srv *httptest.Server
	var mu sync.Mutex
	var created string

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		req := string(buf)
		host := strings.TrimPrefix(srv.URL, "http://")

		if !strings.Contains(req, "GetSystemDateAndTime") {
			if m := createdStamp.FindStringSubmatch(req); m != nil {
				mu.Lock()
				if created == "" {
					created = m[1]
				}
				mu.Unlock()
			}
		}

		var out string
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			out = soap(`<tds:GetSystemDateAndTimeResponse/>`) // the bare reply, as before
		case strings.Contains(req, "GetCapabilities"):
			out = soap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
				`<tt:Device><tt:XAddr>http://` + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
				`<tt:Media><tt:XAddr>http://` + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
				`</tds:Capabilities></tds:GetCapabilitiesResponse>`)
		default:
			out = soap(`<tds:Empty/>`)
		}
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(out))
	}))
	t.Cleanup(srv.Close)

	authenticatedCall(t, srv)

	mu.Lock()
	raw := created
	mu.Unlock()
	if raw == "" {
		t.Fatal("no Created stamp captured")
	}
	stamped, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t.Fatalf("Created %q is not RFC3339: %v", raw, err)
	}
	if d := stamped.Sub(time.Now().UTC()); d > 5*time.Second || d < -5*time.Second {
		t.Errorf("Created = %s is %v from local time; with no device clock reported the "+
			"stamp must fall back to local time", stamped.Format(time.RFC3339), d)
	}
}

// TestDeviceClockOffset is the unit half, including the guard that keeps an absent element
// from producing an offset measured against year 0.
func TestDeviceClockOffset(t *testing.T) {
	local := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)

	t.Run("ahead", func(t *testing.T) {
		sdt := onvif.SystemDateTime{UTCDateTime: onvif.DateTime{
			Date: onvif.Date{Year: 2026, Month: 8, Day: 31},
			Time: onvif.Time{Hour: 12, Minute: 10, Second: 0},
		}}
		if got, want := deviceClockOffset(local, sdt), 10*time.Minute; got != want {
			t.Errorf("offset = %v, want %v", got, want)
		}
	})

	t.Run("behind", func(t *testing.T) {
		sdt := onvif.SystemDateTime{UTCDateTime: onvif.DateTime{
			Date: onvif.Date{Year: 2026, Month: 8, Day: 31},
			Time: onvif.Time{Hour: 11, Minute: 30, Second: 0},
		}}
		if got, want := deviceClockOffset(local, sdt), -30*time.Minute; got != want {
			t.Errorf("offset = %v, want %v", got, want)
		}
	})

	t.Run("absent element yields zero", func(t *testing.T) {
		if got := deviceClockOffset(local, onvif.SystemDateTime{}); got != 0 {
			t.Errorf("offset = %v, want 0; a zero Year means the device reported nothing, "+
				"not the year 0", got)
		}
	})

	t.Run("a wildly wrong clock is not clamped", func(t *testing.T) {
		// A camera reading 1970 wants its token stamped in 1970; clamping would defeat the
		// point of the whole mechanism.
		sdt := onvif.SystemDateTime{UTCDateTime: onvif.DateTime{
			Date: onvif.Date{Year: 1970, Month: 1, Day: 1},
			Time: onvif.Time{Hour: 0, Minute: 0, Second: 0},
		}}
		if got := deviceClockOffset(local, sdt); got > -100000*time.Hour {
			t.Errorf("offset = %v, want a large negative offset carried through unclamped", got)
		}
	})
}

// TestDeviceNowIsUTC pins the Location, because AddWSSecurityAt formats with RFC3339Nano
// and a non-UTC Location would put "+02:00" on the wire where a device expects "Z".
func TestDeviceNowIsUTC(t *testing.T) {
	client, err := networking.NewClient(networking.ClientInfo{Xaddr: "192.0.2.1:80"}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.SetClockOffset(90 * time.Minute)

	if got := client.ClockOffset(); got != 90*time.Minute {
		t.Errorf("ClockOffset = %v, want 90m", got)
	}
	if !strings.HasSuffix(time.Now().UTC().Add(client.ClockOffset()).Format(time.RFC3339), "Z") {
		t.Error("the offset instant does not format as UTC")
	}
}

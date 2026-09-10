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

// AGENTS.md: "Independent fetches against one camera, or probes across interfaces, should
// not be sequential: each costs a network round trip and the CLI runs under a one-minute
// context." Only profiles.go followed that; every Fetch* in device.go, media.go, ptz.go and
// event.go issued its calls one after another, and FetchMediaProfiles walked its profiles
// in sequence at some thirteen round trips each.
//
// Two things have to hold after the fan-out, and each has a test here: the calls really do
// overlap, and the results are still assembled correctly. The second matters more --
// concurrency that returns the wrong answer faster is not an improvement -- and under
// `go test -race`, which CI now runs, these are also the entry point for the race detector.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/xsd/onvif"
)

// replyDelay is what makes concurrency observable: long enough that a sequential run is
// unmistakably slower, short enough to keep the suite quick.
const replyDelay = 40 * time.Millisecond

// concurrencyStub answers every operation after replyDelay, tracks how many requests are in
// flight at once, and counts requests per operation.
type concurrencyStub struct {
	srv *httptest.Server

	inFlight atomic.Int32
	peak     atomic.Int32

	mu     sync.Mutex
	counts map[string]int
}

func (s *concurrencyStub) count(operation string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[operation]
}

// newConcurrencyStub stands up a camera whose replies are supplied by body, which receives
// the request text and the stub's own host.
func newConcurrencyStub(t *testing.T, body func(req, host string) string) *concurrencyStub {
	t.Helper()

	stub := &concurrencyStub{counts: map[string]int{}}

	stub.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := stub.inFlight.Add(1)
		for {
			peak := stub.peak.Load()
			if now <= peak || stub.peak.CompareAndSwap(peak, now) {
				break
			}
		}
		defer stub.inFlight.Add(-1)

		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		req := string(buf)

		// The probe and the capability exchange happen before any fan-out, so they must not
		// be delayed or counted -- otherwise every test pays for them twice.
		probe := strings.Contains(req, "GetSystemDateAndTime") || strings.Contains(req, "GetCapabilities")
		if !probe {
			stub.mu.Lock()
			for _, op := range knownOperations {
				if strings.Contains(req, "<tds:"+op+"/>") || strings.Contains(req, "<tds:"+op+">") ||
					strings.Contains(req, "<trt:"+op+"/>") || strings.Contains(req, "<trt:"+op+">") {
					stub.counts[op]++
					break
				}
			}
			stub.mu.Unlock()
			time.Sleep(replyDelay)
		}

		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(body(req, strings.TrimPrefix(stub.srv.URL, "http://"))))
	}))
	t.Cleanup(stub.srv.Close)

	return stub
}

// knownOperations are the ones these tests count. Kept short on purpose: a longer list
// would make the substring match below ambiguous.
var knownOperations = []string{
	"GetVideoSourceConfigurations", "GetAudioSourceConfigurations",
	"GetVideoSources", "GetAudioSources", "GetProfile", "GetProfiles",
}

func (s *concurrencyStub) profileS(t *testing.T) *ProfileS {
	t.Helper()

	dev, err := NewDevice(context.Background(),
		networking.ClientInfo{Xaddr: strings.TrimPrefix(s.srv.URL, "http://")},
		networking.ClientAuth{Username: "admin", Password: "admin"},
		nil)
	if err != nil {
		t.Fatalf("NewDevice: %v", err)
	}
	profileS, ok := dev.ProfileS()
	if !ok {
		t.Fatal("the stub advertises Device and Media, so Profile S should be available")
	}
	return profileS
}

// capabilities is the reply that puts Device, Media and PTZ in the endpoint map.
func capabilities(host string) string {
	return soap(`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
		`<tt:Device><tt:XAddr>http://` + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
		`<tt:Media><tt:XAddr>http://` + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
		`<tt:PTZ><tt:XAddr>http://` + host + `/onvif/ptz_service</tt:XAddr></tt:PTZ>` +
		`</tds:Capabilities></tds:GetCapabilitiesResponse>`)
}

// TestFetchDeviceSystemIssuesItsCallsConcurrently is the timing half. FetchDeviceSystem
// makes nine independent round trips, so in sequence it costs 9*replyDelay.
func TestFetchDeviceSystemIssuesItsCallsConcurrently(t *testing.T) {
	stub := newConcurrencyStub(t, func(req, host string) string {
		if strings.Contains(req, "GetCapabilities") {
			return capabilities(host)
		}
		if strings.Contains(req, "GetSystemDateAndTime") {
			return soap(`<tds:GetSystemDateAndTimeResponse/>`)
		}
		return soap(`<tds:Empty/>`)
	})

	profileS := stub.profileS(t)

	start := time.Now()
	_ = profileS.FetchDeviceSystem(context.Background())
	elapsed := time.Since(start)

	const calls = 9
	if peak := stub.peak.Load(); peak < 2 {
		t.Errorf("peak in-flight requests = %d; the calls are still sequential", peak)
	}
	// Generous: the point is to separate "overlapping" from "one after another", not to
	// assert a particular degree of parallelism on a loaded CI box.
	if budget := calls * replyDelay / 2; elapsed > budget {
		t.Errorf("FetchDeviceSystem took %v, more than half the sequential cost of %v; "+
			"the fan-out is not taking effect", elapsed, calls*replyDelay)
	}
}

// TestFetchMediaProfilesPopulatesEveryProfile is the correctness half of 9b: the map is
// pre-seeded and then written through pointers, so every profile must come back present and
// distinct. Under -race this is also what exercises that map.
func TestFetchMediaProfilesPopulatesEveryProfile(t *testing.T) {
	tokens := []string{"Profile_1", "Profile_2", "Profile_3", "Profile_4"}

	stub := newConcurrencyStub(t, func(req, host string) string {
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			return soap(`<tds:GetSystemDateAndTimeResponse/>`)
		case strings.Contains(req, "GetCapabilities"):
			return capabilities(host)
		case strings.Contains(req, "GetProfiles"):
			var b strings.Builder
			for _, tok := range tokens {
				b.WriteString(`<tt:Profiles token="` + tok + `"></tt:Profiles>`)
			}
			return soap(`<trt:GetProfilesResponse>` + b.String() + `</trt:GetProfilesResponse>`)
		default:
			return soap(`<tds:Empty/>`)
		}
	})

	profiles := stub.profileS(t).FetchMediaProfiles(context.Background())

	if len(profiles.Profiles) != len(tokens) {
		t.Fatalf("got %d profiles, want %d", len(profiles.Profiles), len(tokens))
	}
	for _, tok := range tokens {
		entry, found := profiles.Profiles[onvifToken(tok)]
		if !found {
			t.Errorf("profile %s is missing", tok)
			continue
		}
		if entry == nil {
			t.Errorf("profile %s has a nil entry", tok)
		}
	}

	// Each profile is fetched exactly once: pre-seeding must not launch a goroutine twice
	// for the same token.
	if n := stub.count("GetProfile"); n != len(tokens) {
		t.Errorf("GetProfile was issued %d times for %d profiles", n, len(tokens))
	}
}

// TestSourceConfigurationsAreFetchedOnce pins the hoist. GetVideoSourceConfigurations takes
// no argument and returns every configuration of the service, yet it was called once per
// video source, storing the identical list on each -- N round trips for one answer.
func TestSourceConfigurationsAreFetchedOnce(t *testing.T) {
	stub := newConcurrencyStub(t, func(req, host string) string {
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			return soap(`<tds:GetSystemDateAndTimeResponse/>`)
		case strings.Contains(req, "GetCapabilities"):
			return capabilities(host)
		case strings.Contains(req, "GetVideoSources"):
			return soap(`<trt:GetVideoSourcesResponse>` +
				`<trt:VideoSources token="VS_1"></trt:VideoSources>` +
				`<trt:VideoSources token="VS_2"></trt:VideoSources>` +
				`<trt:VideoSources token="VS_3"></trt:VideoSources>` +
				`</trt:GetVideoSourcesResponse>`)
		case strings.Contains(req, "GetVideoSourceConfigurations"):
			return soap(`<trt:GetVideoSourceConfigurationsResponse>` +
				`<trt:Configurations token="VSC_1"></trt:Configurations>` +
				`</trt:GetVideoSourceConfigurationsResponse>`)
		default:
			return soap(`<tds:Empty/>`)
		}
	})

	video := stub.profileS(t).FetchMediaVideo(context.Background())

	if len(video.Sources) != 3 {
		t.Fatalf("got %d video sources, want 3", len(video.Sources))
	}
	if n := stub.count("GetVideoSourceConfigurations"); n != 1 {
		t.Errorf("GetVideoSourceConfigurations was issued %d times, want 1; it takes no "+
			"argument and returns every configuration, so once is enough", n)
	}
	// The hoist must not have cost the data: every source still carries the list.
	for i, src := range video.Sources {
		if len(src.Configurations) != 1 {
			t.Errorf("source %d carries %d configurations, want 1", i, len(src.Configurations))
		}
	}
}

// TestFetchMediaAudioOutputsStayConsistent is the guard on the one pair of calls that could
// not be fanned out. GetAudioOutputs fills Audio.Outputs and GetAudioOutputConfigurations
// reads back what it wrote to attach each configuration to its output, so they share one
// goroutine and one order. Splitting them would be a map race and would also lose the
// association -- which is the defect 45bbd6a fixed.
//
// This test fails if someone gives them a goroutine each, whether or not the race detector
// happens to catch the map access on that run.
func TestFetchMediaAudioOutputsStayConsistent(t *testing.T) {
	stub := newConcurrencyStub(t, func(req, host string) string {
		switch {
		case strings.Contains(req, "GetSystemDateAndTime"):
			return soap(`<tds:GetSystemDateAndTimeResponse/>`)
		case strings.Contains(req, "GetCapabilities"):
			return capabilities(host)
		case strings.Contains(req, "GetAudioOutputConfigurations"):
			// OutputToken ties the configuration to the output announced below.
			return soap(`<trt:GetAudioOutputConfigurationsResponse>` +
				`<trt:Configurations token="AOC_1">` +
				`<tt:OutputToken>AO_1</tt:OutputToken>` +
				`</trt:Configurations>` +
				`</trt:GetAudioOutputConfigurationsResponse>`)
		case strings.Contains(req, "GetAudioOutputs"):
			return soap(`<trt:GetAudioOutputsResponse>` +
				`<trt:AudioOutputs token="AO_1"></trt:AudioOutputs>` +
				`</trt:GetAudioOutputsResponse>`)
		default:
			return soap(`<tds:Empty/>`)
		}
	})

	audio := stub.profileS(t).FetchMediaAudio(context.Background())

	if len(audio.Outputs) != 1 {
		t.Fatalf("got %d audio outputs, want 1; a second entry means the configuration "+
			"failed to find the output it belongs to", len(audio.Outputs))
	}
	output, found := audio.Outputs[onvifToken("AO_1")]
	if !found {
		t.Fatal("audio output AO_1 is missing")
	}
	if string(output.Output.Token) != "AO_1" {
		t.Errorf("the entry carries Output.Token = %q; GetAudioOutputs did not fill it, so "+
			"the configuration created the entry first", output.Output.Token)
	}
	if len(output.Configurations) != 1 {
		t.Fatalf("output AO_1 carries %d configurations, want 1", len(output.Configurations))
	}
	if string(output.Configurations[0].Token) != "AOC_1" {
		t.Errorf("configuration = %q, want AOC_1", output.Configurations[0].Token)
	}
}

// onvifToken is the one cast these tests need; spelled out so the import list stays short.
func onvifToken(s string) onvif.ReferenceToken { return onvif.ReferenceToken(s) }

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

// `subscribe` is the only command whose normal end is a signal, and the decisions that
// follow from that are the ones pinned here.
//
// Cancellation used to reach Logger.Fatal at the bottom of main() and exit 1, so
// `onvif-cli subscribe ... > events.jsonl` could not be stopped without the shell reporting
// a failure -- which makes it unusable in a pipeline or a systemd unit. And the one-minute
// deadline used to wrap the whole process, so a streaming command would have ended at one
// minute, silently and successfully.
//
// Like interfaces_test.go and signal_test.go, these pin operator-facing properties of the
// tool rather than a wire format. This is also the only concurrency in the CLI, so it is the
// entry point for the race detector CI runs.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
)

// noDelay collapses the back-off so a retry test is not a stopwatch. streamCamera takes the
// delay as a parameter for that reason -- the same seam, and the same argument, as prober in
// discover.go.
func noDelay(int) time.Duration { return 0 }

var oneCamera = []networking.ClientInfo{{Xaddr: "192.168.1.70:80"}}

// fakeStream is one camera's subscription, answered by a closure.
type fakeStream struct {
	pull func(ctx context.Context, call int) ([]sdk.Notification, error)

	mu       sync.Mutex
	pulls    int
	released int
}

func (f *fakeStream) Pull(ctx context.Context) ([]sdk.Notification, error) {
	f.mu.Lock()
	f.pulls++
	call := f.pulls
	f.mu.Unlock()
	return f.pull(ctx, call)
}

func (f *fakeStream) Unsubscribe(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.released++
	return nil
}

func (f *fakeStream) releases() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.released
}

// lines splits a stream of records, dropping the trailing newline.
func lines(out string) []string {
	out = strings.TrimSuffix(out, "\n")
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// An empty stdout says the same thing as a quiet fleet, so a run that subscribed to nothing
// cannot be a success -- and a run that subscribed to some of its fleet cannot be a failure,
// or the exit status becomes a report on the network weather of an arbitrary instant.
func TestNothingSubscribedIsAFailure(t *testing.T) {
	for _, tc := range []struct {
		name        string
		established int
		targets     int
		wantErr     bool
	}{
		{"the only camera named could not be subscribed", 0, 1, true},
		{"none of a whole fleet could", 0, 5, true},
		{"one of five is enough to stream", 1, 5, false},
		{"and so is all five", 5, 5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := establishError(tc.established, tc.targets)
			if tc.wantErr {
				if err == nil {
					t.Fatal("a run that subscribed to nothing reported success, so an empty " +
						"stream is indistinguishable from a quiet fleet")
				}
				if !strings.Contains(err.Error(), fmt.Sprint(tc.targets)) {
					t.Errorf("the error does not say how many targets were named: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("establishError(%d, %d) = %v, want nil", tc.established, tc.targets, err)
			}
		})
	}
}

// Stopping the run is how a subscription ends, so it is a success. The second assertion is
// the other half: a retry loop that does not consult the context first turns a shutdown into
// a hot loop of failing resubscriptions, and the error from an interrupted pull wraps
// context.Canceled exactly as a genuine one might.
func TestStoppingIsNotAFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var opens atomic.Int32
	open := func(_ context.Context, dev networking.ClientInfo) (stream, error) {
		opens.Add(1)
		return &fakeStream{pull: func(pull context.Context, call int) ([]sdk.Notification, error) {
			if call == 1 {
				return []sdk.Notification{crossing}, nil
			}
			// The operator's Ctrl-C, once one record is out.
			cancel()
			<-pull.Done()
			// The shape net/http really produces.
			return nil, fmt.Errorf("Post %q: %w", "http://cam/Subscription", context.Canceled)
		}}, nil
	}

	var out bytes.Buffer
	if err := streamFleet(ctx, oneCamera, &out, open, noDelay); err != nil {
		t.Fatalf("streamFleet = %v, want nil — an interrupted run must exit 0, or it cannot "+
			"be used in a pipeline or a unit", err)
	}
	if got := len(lines(out.String())); got != 1 {
		t.Errorf("got %d records, want the one written before the stop", got)
	}
	if got := opens.Load(); got != 1 {
		t.Errorf("the camera was subscribed %d times; a stop was retried as though it were "+
			"a dropped pull point", got)
	}
}

// A device failure arriving in the same instant as the signal does not change the exit
// status. Once a stop has been asked for there is no evidence left inside the process that
// could tell the two apart, so the only defensible reading is that the run still succeeded;
// the diagnostic and the status are separate, and the error is on stderr at debug.
func TestAFailureWhileStoppingIsNotFatal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	open := func(_ context.Context, dev networking.ClientInfo) (stream, error) {
		return &fakeStream{pull: func(pull context.Context, call int) ([]sdk.Notification, error) {
			if call == 1 {
				return []sdk.Notification{crossing}, nil
			}
			cancel()
			// Not a cancellation at all: a fault, which is what makes this the coincidence
			// case rather than the ordinary one above.
			return nil, errors.New("500 Internal Server Error")
		}}, nil
	}

	var out bytes.Buffer
	if err := streamFleet(ctx, oneCamera, &out, open, noDelay); err != nil {
		t.Errorf("streamFleet = %v, want nil", err)
	}
}

// There is no Renew: ONVIF Core section 9.1 has the device extend the subscription on every
// PullMessages, so the only failure left to handle is a pull point that has gone, and the
// answer to that is a new subscription.
func TestALostSubscriptionIsResubscribed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opens := 0
	open := func(_ context.Context, dev networking.ClientInfo) (stream, error) {
		opens++
		if opens == 1 {
			// The pull point is gone, while the run is very much alive.
			return &fakeStream{pull: func(context.Context, int) ([]sdk.Notification, error) {
				return nil, errors.New("http request error: PullMessages: 500 Internal Server Error")
			}}, nil
		}
		return &fakeStream{pull: func(pull context.Context, call int) ([]sdk.Notification, error) {
			if call == 1 {
				return []sdk.Notification{crossing}, nil
			}
			cancel()
			<-pull.Done()
			return nil, pull.Err()
		}}, nil
	}

	var out bytes.Buffer
	if err := streamFleet(ctx, oneCamera, &out, open, noDelay); err != nil {
		t.Fatalf("streamFleet = %v, want nil", err)
	}
	if opens != 2 {
		t.Errorf("the camera was subscribed %d times, want 2 — a dropped pull point was not "+
			"recovered, so the command stops watching a camera that is still there", opens)
	}
	if got := len(lines(out.String())); got != 1 {
		t.Errorf("got %d records, want the one the new subscription delivered", got)
	}
}

// Both pull points are released, and the reason is that a failed pull does not establish
// that the pull point is gone. The commonest cause is one exchange that timed out on a busy
// device, which leaves the subscription alive -- and an orphan lives until its termination
// time, while event.Capabilities.MaxPullPoints is one to four on real firmware. A few
// network blips would then exhaust the device and make the next resubscription fault in a way
// that reads exactly like a rejected credential.
//
// Unsubscribe is bounded and is already a no-op against a pull point the device has
// forgotten, so the round trip that "can only fail" costs at most unsubscribeTimeout, and
// only on a camera that has just gone quiet anyway. Verified against ONVIF Core sections
// 9.1.4 and 9.1.6, the latter capping pull points at the advertised MaxPullPoints.
func TestBothALostAndALivePullPointAreReleased(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lost := &fakeStream{pull: func(context.Context, int) ([]sdk.Notification, error) {
		// What a slow camera really produces: the client's own Timeout, and not a fault
		// saying the subscription is gone.
		return nil, fmt.Errorf("Post %q: %w", "http://cam/Subscription",
			errors.New("context deadline exceeded (Client.Timeout exceeded)"))
	}}
	live := &fakeStream{pull: func(pull context.Context, call int) ([]sdk.Notification, error) {
		cancel()
		<-pull.Done()
		return nil, pull.Err()
	}}

	opens := 0
	open := func(_ context.Context, dev networking.ClientInfo) (stream, error) {
		opens++
		if opens == 1 {
			return lost, nil
		}
		return live, nil
	}

	var out bytes.Buffer
	if err := streamFleet(ctx, oneCamera, &out, open, noDelay); err != nil {
		t.Fatalf("streamFleet = %v, want nil", err)
	}
	if got := lost.releases(); got != 1 {
		t.Errorf("the pull point that stopped answering was released %d times, want 1 — it "+
			"is otherwise orphaned on the device for its whole termination time, and a "+
			"device with MaxPullPoints=2 faults the third resubscription", got)
	}
	if got := live.releases(); got != 1 {
		t.Errorf("the live pull point was released %d times, want 1 — a device with "+
			"MaxPullPoints=1 then refuses the next run until this one expires", got)
	}
}

// A camera that is rebooting takes tens of seconds, so a zero or ungrowing delay is both a
// flood of stderr and a small denial of service against a device that accepts a handful of
// connections at a time.
func TestResubscribeDelayIsBoundedAndNeverZero(t *testing.T) {
	previous := time.Duration(0)
	for attempt := range 20 {
		got := resubscribeDelay(attempt)

		if got <= 0 {
			t.Fatalf("resubscribeDelay(%d) = %v; attempt 1 is already a retry, and something "+
				"has just failed", attempt, got)
		}
		if got < resubscribeDelayMin {
			t.Errorf("resubscribeDelay(%d) = %v, below the floor %v", attempt, got, resubscribeDelayMin)
		}
		if got > resubscribeDelayMax {
			t.Errorf("resubscribeDelay(%d) = %v, above the cap %v — a camera that stays down "+
				"would be waited for longer and longer without bound", attempt, got, resubscribeDelayMax)
		}
		if got < previous {
			t.Errorf("resubscribeDelay(%d) = %v, less than the previous %v", attempt, got, previous)
		}
		previous = got
	}
	if resubscribeDelay(20) != resubscribeDelayMax {
		t.Errorf("the delay never reaches its cap, so it is not really bounded by it")
	}
}

// json.Encoder is not safe for concurrent use, and a record carrying an ElementItem's XML
// exceeds a pipe's atomic write size, so a torn write is not theoretical. Every line has to
// be whole and every record has to arrive.
func TestConcurrentCamerasDoNotInterleaveStdout(t *testing.T) {
	const (
		cameras   = 32
		perCamera = 50
		wantTotal = cameras * perCamera
	)

	cams := make([]networking.ClientInfo, cameras)
	for i := range cams {
		cams[i] = networking.ClientInfo{Xaddr: fmt.Sprintf("10.0.0.%d:80", i+1)}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var emitted atomic.Int32
	open := func(_ context.Context, dev networking.ClientInfo) (stream, error) {
		sent := 0
		return &fakeStream{pull: func(pull context.Context, call int) ([]sdk.Notification, error) {
			if sent >= perCamera {
				<-pull.Done()
				return nil, pull.Err()
			}
			sent++

			notification := crossing
			// One oversized record per camera, so a torn write tears rather than happening
			// to fit inside PIPE_BUF.
			if sent == 1 {
				notification.Data = []sdk.Item{{Name: "Blob", Value: strings.Repeat("x", 8<<10)}}
			}
			if emitted.Add(1) == wantTotal {
				cancel()
			}
			return []sdk.Notification{notification}, nil
		}}, nil
	}

	var out bytes.Buffer
	if err := streamFleet(ctx, cams, &out, open, noDelay); err != nil {
		t.Fatalf("streamFleet = %v, want nil", err)
	}

	got := lines(out.String())
	if len(got) != wantTotal {
		t.Fatalf("got %d records, want %d — records were lost or spliced", len(got), wantTotal)
	}

	perXaddr := map[string]int{}
	for i, line := range got {
		var record struct {
			Xaddr string `json:"xaddr"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("line %d does not parse on its own, so two records were interleaved: %v", i, err)
		}
		perXaddr[record.Xaddr]++
	}
	if len(perXaddr) != cameras {
		t.Errorf("%d cameras appear in the output, want %d", len(perXaddr), cameras)
	}
	for xaddr, count := range perXaddr {
		if count != perCamera {
			t.Errorf("%s wrote %d records, want %d", xaddr, count, perCamera)
		}
	}
}

// PullMessages is a long poll by design, and one pull is ONE HTTP exchange -- so the client's
// own Timeout is the real ceiling on the poll window. A window at or above it is cut off on
// our side every single round, and a camera that works perfectly looks like one that never
// answers. The two numbers are declared in different files and nothing else relates them.
func TestPullTimeoutFitsTheHTTPClientTimeout(t *testing.T) {
	if pullTimeout >= httpClient.Timeout {
		t.Fatalf("pullTimeout %v is not below the client Timeout %v; every idle pull would "+
			"be aborted locally", pullTimeout, httpClient.Timeout)
	}
	// And with room for the round trip itself, not merely by a hair.
	if margin := httpClient.Timeout - pullTimeout; margin < 5*time.Second {
		t.Errorf("only %v separates the poll window from the client timeout, which leaves "+
			"nothing for the exchange around it", margin)
	}
}

// The deadline belongs to the commands that have an end of their own. A stray WithTimeout
// anywhere above the pull would cap the stream at it, which is a bug that reads as a firmware
// defect.
func TestStreamingCarriesNoDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	open := func(establishing context.Context, dev networking.ClientInfo) (stream, error) {
		// Establishing one camera IS bounded, and should be: past a minute the camera is
		// not slow, it is absent.
		if _, ok := establishing.Deadline(); !ok {
			t.Error("establishing a subscription carries no deadline, so an absent camera " +
				"holds its slot for as long as the transport allows")
		}
		return &fakeStream{pull: func(pull context.Context, call int) ([]sdk.Notification, error) {
			if _, ok := pull.Deadline(); ok {
				t.Error("the pull carries a deadline, so the subscription would end at it — " +
					"silently, and reported as a success")
			}
			cancel()
			<-pull.Done()
			return nil, pull.Err()
		}}, nil
	}

	var out bytes.Buffer
	if err := streamFleet(ctx, oneCamera, &out, open, noDelay); err != nil {
		t.Fatalf("streamFleet = %v, want nil", err)
	}
}

// The other half of the same decision: a one-shot command still gets its minute.
func TestRunOneShotBoundsItsCallee(t *testing.T) {
	called := false
	err := runOneShot(context.Background(), func(ctx context.Context) error {
		called = true
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("runOneShot did not bound its callee, so `dump` and `discover` would " +
				"wait on a stuck camera forever")
		}
		if remaining := time.Until(deadline); remaining > oneShotDeadline {
			t.Errorf("the deadline is %v away, more than oneShotDeadline %v", remaining, oneShotDeadline)
		}
		return nil
	})
	if err != nil {
		t.Errorf("runOneShot = %v", err)
	}
	if !called {
		t.Error("runOneShot did not call its argument")
	}
}

// A camera nobody could subscribe to is named and left out, and the others still stream --
// the shape `streams` already has, where a per-camera failure is a log line and not the end
// of the run.
func TestOneDeadCameraDoesNotStopTheFleet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cams := []networking.ClientInfo{
		{Xaddr: "192.168.1.70:80"},
		{Xaddr: "192.168.1.71:80"},
	}

	open := func(_ context.Context, dev networking.ClientInfo) (stream, error) {
		if dev.Xaddr == "192.168.1.71:80" {
			return nil, ErrNoEventService
		}
		return &fakeStream{pull: func(pull context.Context, call int) ([]sdk.Notification, error) {
			if call == 1 {
				return []sdk.Notification{crossing}, nil
			}
			cancel()
			<-pull.Done()
			return nil, pull.Err()
		}}, nil
	}

	var out bytes.Buffer
	if err := streamFleet(ctx, cams, &out, open, noDelay); err != nil {
		t.Fatalf("streamFleet = %v, want nil — one camera without an event service must not "+
			"fail a fleet command", err)
	}
	if got := len(lines(out.String())); got != 1 {
		t.Errorf("got %d records, want the one the reachable camera sent", got)
	}
}

// And when none of them could be subscribed, the run says so at once rather than leaving an
// operator waiting for events that could never arrive.
func TestAFleetThatSubscribedToNothingFails(t *testing.T) {
	open := func(_ context.Context, dev networking.ClientInfo) (stream, error) {
		return nil, ErrNoEventService
	}

	var out bytes.Buffer
	err := streamFleet(context.Background(), oneCamera, &out, open, noDelay)
	if err == nil {
		t.Fatal("streamFleet reported success having subscribed to nothing")
	}
	if out.Len() != 0 {
		t.Errorf("the failing run still wrote to stdout: %q", out.String())
	}
}

// Two decisions meet when the operator interrupts while cameras are still being connected:
// "exit 1 when no camera could be subscribed" and "an interrupt is a successful end". The
// interrupt wins, because the operator knows why the stream is empty -- they asked -- and
// establishError's message would blame the fleet for their own Ctrl-C.
func TestStoppingWhileSubscribingIsNotAFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	open := func(establishing context.Context, dev networking.ClientInfo) (stream, error) {
		// The Ctrl-C lands mid-connect, which is what a camera behind a dead link gives an
		// impatient operator plenty of time to do.
		cancel()
		<-establishing.Done()
		return nil, fmt.Errorf("connecting to %s: %w", dev.Xaddr, context.Canceled)
	}

	var out bytes.Buffer
	if err := streamFleet(ctx, oneCamera, &out, open, noDelay); err != nil {
		t.Errorf("streamFleet = %v, want nil — an interrupt is the successful end of this "+
			"command wherever it lands, and this one reads as a fleet-wide failure", err)
	}
	if out.Len() != 0 {
		t.Errorf("the interrupted run wrote to stdout: %q", out.String())
	}
}

// `discover` prints one line per interface a device answered on, so the pipeline the
// Example documents hands a dual-homed camera over twice -- and two pull points on one
// camera doubles every event on stdout with nothing to say why.
func TestDuplicateTargetsAreSubscribedOnce(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []networking.ClientInfo
		want []networking.ClientInfo
	}{
		{
			"the same camera twice, as a dual-homed host makes discover print it",
			[]networking.ClientInfo{
				{Xaddr: "192.168.1.70:80", Uuid: "urn:uuid:a"},
				{Xaddr: "192.168.1.70:80", Uuid: "urn:uuid:a"},
			},
			[]networking.ClientInfo{{Xaddr: "192.168.1.70:80", Uuid: "urn:uuid:a"}},
		},
		{
			// The identifier is the only form that selects a per-camera credentials file, so
			// keeping the address-only entry alone would quietly downgrade the camera to the
			// blanket credentials.
			"named by address first and by identifier second keeps the identifier",
			[]networking.ClientInfo{
				{Xaddr: "192.168.1.70:80"},
				{Xaddr: "192.168.1.70:80", Uuid: "urn:uuid:a"},
			},
			[]networking.ClientInfo{{Xaddr: "192.168.1.70:80", Uuid: "urn:uuid:a"}},
		},
		{
			"distinct cameras are all kept, in the order the operator named them",
			[]networking.ClientInfo{
				{Xaddr: "192.168.1.71:80"},
				{Xaddr: "192.168.1.70:80"},
			},
			[]networking.ClientInfo{{Xaddr: "192.168.1.71:80"}, {Xaddr: "192.168.1.70:80"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := dedupeTargets(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("dedupeTargets returned %d targets, want %d: %+v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("target %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// A device that ignores tev:Timeout and answers an empty reply at once must not be pulled in
// a tight loop, and a device that has an event for us must not be delayed by a millisecond.
// ONVIF Core section 9.1.2 -- "the device shall not return a response with zero messages
// prior to reaching the Timeout" -- is what the fast path assumes, and this is what covers
// the firmware where the assumption does not hold.
func TestPullThrottle(t *testing.T) {
	for _, tc := range []struct {
		name     string
		messages int
		elapsed  time.Duration
		want     time.Duration
	}{
		{"a reply carrying events is never delayed", 3, 0, 0},
		{"nor is one that carried events after a long poll", 1, 9 * time.Second, 0},
		{"an instant empty reply waits out the floor", 0, 0, minPullInterval},
		{"a nearly instant one waits out the remainder", 0, 900 * time.Millisecond, 100 * time.Millisecond},
		{"a long poll that expired empty is the documented case and waits for nothing", 0, pullTimeout, 0},
		{"and one exactly at the floor likewise", 0, minPullInterval, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := pullThrottle(tc.messages, tc.elapsed); got != tc.want {
				t.Errorf("pullThrottle(%d, %v) = %v, want %v", tc.messages, tc.elapsed, got, tc.want)
			}
		})
	}

	// The floor has to sit below the poll window, or it would throttle the ordinary idle
	// case that section 9.1.2 entitles a device to.
	if minPullInterval >= pullTimeout {
		t.Errorf("the floor %v is not below the poll window %v", minPullInterval, pullTimeout)
	}
}

// `discover` prints one line per interface per device, so the pipeline this command
// documents in its Example feeds a dual-homed camera in twice. The address form never
// probes, so the whole policy is testable without a LAN.
func TestATargetNamedTwiceIsSubscribedOnce(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{"one camera is one target", []string{"192.168.1.70:80"}, []string{"192.168.1.70:80"}},
		{
			"the same address twice is still one camera",
			[]string{"192.168.1.70:80", "192.168.1.70:80"},
			[]string{"192.168.1.70:80"},
		},
		{
			"order is the operator's, first mention winning",
			[]string{"192.168.1.71:80", "192.168.1.70:80", "192.168.1.71:80"},
			[]string{"192.168.1.71:80", "192.168.1.70:80"},
		},
		{
			"two cameras stay two",
			[]string{"192.168.1.70:80", "192.168.1.71:80"},
			[]string{"192.168.1.70:80", "192.168.1.71:80"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveTargets(context.Background(), tc.args)
			if err != nil {
				t.Fatalf("resolveTargets: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("resolveTargets(%v) returned %d targets, want %d", tc.args, len(got), len(tc.want))
			}
			for i := range got {
				if got[i].Xaddr != tc.want[i] {
					t.Errorf("target %d = %q, want %q", i, got[i].Xaddr, tc.want[i])
				}
			}
		})
	}
}

// A pull point created before the operator's Ctrl-C landed is a pull point the device is
// still holding.
//
// streamFleet returned from its "stopped while subscribing" branch with every established
// subscription still sitting in `opened`, releasing none of them: the only Unsubscribe in
// this command is the defer inside streamCamera, and the streaming loop that reaches it is
// below that return. It even logs how many are open on the way past.
//
// Neither -race nor the compiler can see this. Every goroutine exits and the memory is
// collected; what leaks is on the camera. sdk.PullPoint.Unsubscribe runs on
// context.WithoutCancel precisely so that it still works from a cancelled path -- see its
// comment on MaxPullPoints being one to four on real firmware, and on the resubscribe fault
// reading like a rejected credential. A supervisor restarting the collector a few times is
// all it takes.
//
// TestStoppingWhileSubscribingIsNotAFailure above looks like it covers this window and does
// not: its open always fails, so no subscription is ever created for it to leak.
func TestSubscriptionsOpenedBeforeTheStopAreReleased(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cams := []networking.ClientInfo{{Xaddr: "192.168.1.70:80"}, {Xaddr: "192.168.1.71:80"}}

	live := &fakeStream{pull: func(context.Context, int) ([]sdk.Notification, error) {
		t.Error("a camera was pulled after the run had been stopped")
		return nil, nil
	}}

	// The two open calls run concurrently, so the interrupt is sequenced behind the first
	// subscription rather than raced against it.
	subscribed := make(chan struct{})
	open := func(establishing context.Context, dev networking.ClientInfo) (stream, error) {
		if dev.Xaddr == cams[0].Xaddr {
			close(subscribed) // this camera's pull point now exists on the device
			return live, nil
		}
		<-subscribed
		cancel()
		<-establishing.Done()
		return nil, fmt.Errorf("connecting to %s: %w", dev.Xaddr, context.Canceled)
	}

	var out bytes.Buffer
	if err := streamFleet(ctx, cams, &out, open, noDelay); err != nil {
		t.Fatalf("streamFleet = %v, want nil -- an interrupt is a successful end", err)
	}
	if got := live.releases(); got != 1 {
		t.Errorf("Unsubscribe called %d times on the camera subscribed before the stop, want 1: "+
			"the device is left holding that pull point until its termination time", got)
	}
	if out.Len() != 0 {
		t.Errorf("the interrupted run wrote to stdout: %q", out.String())
	}
}

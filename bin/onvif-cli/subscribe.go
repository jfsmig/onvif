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

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
)

// subscribeHelp completes targetHelp with what is specific to a fleet command that streams:
// one probe answers however many identifiers are named, and the run ends when the operator
// says so rather than when the work runs out.
const subscribeHelp = `Every identifier named is resolved from a single WS-Discovery probe, so ten cameras cost
one probe and not ten. An identifier no device claims fails the command before anything is
streamed, naming the identifiers that did answer.

One JSON object per line goes to stdout, one line per event, and nothing else does. The run
ends when it is interrupted -- Ctrl-C, or the SIGTERM a supervisor sends -- and that is a
success: a subscription has no other way to end. A camera that drops its pull point is
resubscribed; a camera that never answered at all is named on stderr and left out, and the
command fails only if not one of them could be subscribed.

There is no --all. Unlike ` + "`discover`" + `, this command authenticates to every target, and a
discovery answer is not an authentication -- so name the cameras, or pipe them in from
` + "`discover`" + `.`

const (
	// pullTimeout is the PullMessages long-poll window (ONVIF Core section 9.1.2): the device
	// holds the request open until it has messages or the window expires, which is what makes
	// the pull its own keep-alive and why there is no Renew anywhere here.
	//
	// It MUST stay below httpClient.Timeout above. A pull is ONE HTTP exchange, so a window
	// at or above that limit is cut off on our side every single round, and a camera that is
	// working perfectly then looks like one that never answers -- the most misleading failure
	// this command could produce. Section 9.1.2 requires a device to support at least a
	// minute, so anything under that is safe on the device's side, and a short window costs
	// only an extra round trip while idle: the timeout is an upper bound, and a message that
	// arrives sooner is returned sooner. subscribe_test.go pins the inequality, because the
	// two numbers are declared in different places and nothing else relates them.
	pullTimeout = 10 * time.Second

	// pullMessageLimit caps one reply, not the run: every message pulled is printed. A rule
	// engine firing on consecutive frames should not cost one round trip per event, and the
	// reply is already bounded by networking.MaxResponseBytes. Section 9.1.2 says a device
	// shall not fault on a limit larger than it supports and shall return what it has
	// instead, so this is a request rather than a contract -- which is also why it is not a
	// flag: an operator-set value a device silently caps is a lie.
	pullMessageLimit = 100

	// establishTimeout bounds one attempt at connecting to a camera and creating its pull
	// point -- four exchanges, against a device that may be absent. The same budget a
	// one-shot command gets, and for the same reason: past that the camera is not slow, it
	// is absent. Spelled as oneShotDeadline rather than repeated, so the sentence before
	// this one cannot quietly become false.
	establishTimeout = oneShotDeadline

	// resubscribeDelay* bound the wait between attempts to get a dropped subscription back.
	//
	// A camera that is rebooting takes tens of seconds, so a tight retry loop is both a
	// flood of stderr and a small denial of service against an embedded web server that
	// accepts a handful of connections at a time -- the same device limit httpClient's
	// MaxConnsPerHost is sized for. Doubling from the first, capped at the second, reset by
	// a subscription that came back.
	resubscribeDelayMin = time.Second
	resubscribeDelayMax = 30 * time.Second

	// minPullInterval floors the gap between two pulls of one camera.
	//
	// ONVIF Core section 9.1.2 says a device "shall not return a response with zero messages
	// prior to reaching the Timeout", and pullTimeout leans on that: it is what makes one
	// pull per window the steady state. Firmware that ignores tev:Timeout and answers an
	// empty PullMessagesResponse at once turns this loop into hundreds of authenticated
	// POSTs a second at a web server that accepts four connections -- the same denial of
	// service resubscribeDelayMin exists to prevent, arriving through the success path
	// instead of the failure one.
	//
	// It never delays a reply that carried messages: an event is the whole reason the
	// command exists.
	minPullInterval = time.Second
)

// ErrNoEventService reports a camera that advertises no event service.
//
// Named rather than left to surface as utils.ErrNoService from the first request: the
// appliance said so in its own GetCapabilities reply, so this is knowable before anything is
// sent, and "this camera does not do events" is a different thing for an operator to read
// than a failed operation.
var ErrNoEventService = errors.New("the appliance advertises no ONVIF event service")

// stream is one camera's live subscription, as the fan-out below uses it.
//
// *sdk.PullPoint satisfies it. The interface exists so that the fleet logic -- the retry
// policy, the output discipline, the exit status -- can be exercised without a camera and
// under -race, which is the same seam, and for the same reason, as prober in discover.go.
type stream interface {
	Pull(ctx context.Context) ([]sdk.Notification, error)
	Unsubscribe(ctx context.Context) error
}

// opener establishes one camera's subscription. openPullPoint is the only implementation.
type opener func(ctx context.Context, dev networking.ClientInfo) (stream, error)

// subscribe holds a subscription on every named camera and streams their events until the
// run is stopped.
func subscribe(ctx context.Context, args []string) error {
	// Resolution is the bounded phase -- parsing, and one LAN probe however many identifiers
	// were named -- so it is the only part of this command that carries a deadline.
	var cams []networking.ClientInfo
	err := runOneShot(ctx, func(ctx context.Context) error {
		var err error
		cams, err = resolveTargets(ctx, args)
		return err
	})
	if err != nil {
		return err
	}

	// Streaming runs on the signal context itself, never on the one above: runOneShot
	// cancels its own on the way out, and a subscription created under it would die the
	// instant resolution returned -- which would look exactly like every camera in the
	// fleet going silent at once.
	return streamFleet(ctx, cams, os.Stdout, openPullPoint, resubscribeDelay)
}

// streamFleet subscribes to every camera, then streams them all onto one writer.
//
// sync.WaitGroup and not errgroup, and here the reason AGENTS.md gives is the shape of the
// command: the first camera to fail must not cancel its siblings' subscriptions.
//
// How large a fleet this takes is bounded by the process file-descriptor limit and not by
// anything here. One live camera means one goroutine holding one pull open, so the steady
// state is one socket per camera -- httpClient's MaxConnsPerHost is per host and caps nothing
// across a fleet of distinct hosts. Batching the establish phase would lower its burst and
// leave that steady state exactly where it is, so it is not done; past the limit the failures
// read as unreachable cameras, and `ulimit -n` is the answer.
func streamFleet(ctx context.Context, cams []networking.ClientInfo, out io.Writer,
	open opener, delay func(attempt int) time.Duration) error {
	// The writer holds this cancel, so a stdout that cannot be written to ends the run
	// rather than leaving every camera pulling into a closed file.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	writer := &recordWriter{encoder: json.NewEncoder(out), cancel: cancel}

	// The first round of subscriptions is what the exit status is computed from, and it
	// happens before any camera enters its retry loop: one live pull point is the difference
	// between "the fleet is quiet" and "nothing was ever subscribed", and stdout is empty in
	// both cases. Concurrent, because otherwise eight absent cameras cost eight
	// establishTimeouts one after another before the reachable ones start.
	opened := make([]stream, len(cams))
	var establishing sync.WaitGroup
	for i, dev := range cams {
		establishing.Go(func() { opened[i] = establish(ctx, dev, open) })
	}
	establishing.Wait()

	live := 0
	for _, subscription := range opened {
		if subscription != nil {
			live++
		}
	}
	if ctx.Err() != nil {
		// Stopped while still connecting. Nothing was delivered, and yet this is a success:
		// the operator knows why the stream is empty, because they asked for it, and
		// establishError's "no camera could be subscribed" would blame the fleet for a
		// Ctrl-C. An interrupt is the successful end of this command wherever it lands.
		//
		// What was opened before the signal landed still has to be released. The streaming
		// loop below is what normally does that, through streamCamera's defer, and it is
		// never reached from here -- so a camera that answered CreatePullPointSubscription
		// in the moment before the Ctrl-C was left holding an orphan until its termination
		// time. PullPoint.Unsubscribe runs on context.WithoutCancel for exactly this
		// instant, and MaxPullPoints is one to four on real firmware, so a supervisor
		// restarting this collector exhausts a camera in a handful of runs -- after which
		// CreatePullPointSubscription faults in a way that reads like a bad password.
		var releasing sync.WaitGroup
		for i, dev := range cams {
			if opened[i] == nil {
				continue
			}
			releasing.Go(func() {
				if err := opened[i].Unsubscribe(ctx); err != nil {
					Logger.Debug().Str("addr", dev.Xaddr).Err(err).Msg("pull point not released")
				}
			})
		}
		releasing.Wait()

		Logger.Debug().Int("subscribed", live).Msg("stopped while subscribing")
		return nil
	}
	if err := establishError(live, len(cams)); err != nil {
		return err
	}
	// The window and the limit at info, because they are the answer to "why has nothing
	// appeared yet" on a quiet camera and they are otherwise invisible: neither is a flag.
	Logger.Info().Int("subscribed", live).Int("targets", len(cams)).
		Dur("window", pullTimeout).Int("limit", pullMessageLimit).
		Msg("streaming events until interrupted")

	var streaming sync.WaitGroup
	for i, dev := range cams {
		if opened[i] == nil {
			continue
		}
		streaming.Go(func() { streamCamera(ctx, dev, opened[i], writer, open, delay) })
	}
	streaming.Wait()

	Logger.Info().Int("records", writer.written()).Int("cameras", live).
		Msg("subscription stopped")
	return writer.failure()
}

// establish makes one attempt at subscribing to one camera, and reports it as nil on
// failure.
//
// The per-attempt context is cancelled as soon as the attempt is over, which is safe because
// nothing in the chain below retains it: sdk.NewDevice and PullPoint.Subscribe each use the
// context for their own exchanges and keep no reference, so the subscription that comes back
// outlives it. A type that did retain it would make every camera stop after
// establishTimeout, which is why this is worth saying out loud.
func establish(ctx context.Context, dev networking.ClientInfo, open opener) stream {
	attempt, cancel := context.WithTimeout(ctx, establishTimeout)
	defer cancel()

	subscription, err := open(attempt, dev)
	if err == nil {
		return subscription
	}
	// Silent once a stop has been asked for: every exchange in flight is expected to fail
	// then, and reporting each one would bury the reason the run ended.
	if ctx.Err() == nil {
		Logger.Error().Str("addr", dev.Xaddr).Str("uuid", uuidColumn(dev.Uuid)).Err(err).
			Msg("no subscription on this camera")
	}
	return nil
}

// streamCamera holds one camera's subscription for as long as the run lasts.
//
// There is no Renew: ONVIF Core section 9.1 has the device extend the subscription on every
// PullMessages, so the pull is its own keep-alive and the only failure left to handle is a
// pull point that has gone -- a camera that rebooted, or a device that dropped the
// subscription while nothing was in flight. The answer to that is a new subscription, not a
// Renew that would fail for the same reason.
//
// Whether this camera stopped because the run did is decided by asking the context, never by
// the shape of the error. An interrupted PullMessages comes back as a *url.Error wrapping
// context.Canceled, and so does the ordinary case of a pull the operator happened to
// interrupt, so branching on the shape would retry a shutdown and swallow a real failure
// with equal enthusiasm. An error that arrives in the same instant as the signal is
// therefore logged and changes nothing about the exit status, which is the only defensible
// reading: once a stop has been asked for there is no evidence left inside the process that
// could tell the two apart.
func streamCamera(ctx context.Context, dev networking.ClientInfo, subscription stream,
	out *recordWriter, open opener, delay func(attempt int) time.Duration) {
	// Closed over the variable, so a pull point already known lost is not unsubscribed from:
	// that would be a round trip that can only fail.
	defer func() {
		if subscription == nil {
			return
		}
		if err := subscription.Unsubscribe(ctx); err != nil {
			Logger.Warn().Str("addr", dev.Xaddr).Err(err).Msg("pull point not released")
		}
	}()

	attempt := 0
	for {
		if ctx.Err() != nil {
			return
		}

		if subscription == nil {
			resubscribed := establishAgain(ctx, dev, open, attempt)
			if resubscribed == nil {
				if ctx.Err() != nil {
					return
				}
				attempt++
				if !pause(ctx, delay(attempt)) {
					return
				}
				continue
			}
			subscription = resubscribed
			attempt = 0
			Logger.Info().Str("addr", dev.Xaddr).Msg("resubscribed")
		}

		started := time.Now()
		notifications, err := subscription.Pull(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// At debug, not warn: once a stop has been asked for every exchange in
				// flight is expected to fail, so a fleet of forty cameras would print forty
				// warnings on an ordinary Ctrl-C and bury whatever the operator stopped for.
				// It is still here, because a genuine device failure can arrive in the same
				// instant and this is the only trace of it.
				Logger.Debug().Str("addr", dev.Xaddr).Err(err).Msg("pull interrupted")
				return
			}
			Logger.Warn().Str("addr", dev.Xaddr).Err(err).Msg("pull point lost")
			// Released rather than dropped, and the asymmetry with the deferred call above
			// is the point: a failed pull does not establish that the pull point is gone.
			// The commonest cause is one exchange that timed out on a busy device, which
			// leaves the subscription very much alive -- and an orphan lives until its
			// termination time, while event.Capabilities.MaxPullPoints is one to four on
			// real firmware. Three blips inside a minute then exhaust the device and the
			// resubscription faults in the way sdk's Unsubscribe comment describes, which
			// reads exactly like a rejected credential. Unsubscribe is bounded at
			// unsubscribeTimeout and is already a no-op against a pull point the device has
			// forgotten, so the round trip costs at most that, and only on a camera that
			// has just gone quiet anyway.
			if err := subscription.Unsubscribe(ctx); err != nil {
				Logger.Debug().Str("addr", dev.Xaddr).Err(err).
					Msg("lost pull point not released")
			}
			subscription = nil
			attempt++
			if !pause(ctx, delay(attempt)) {
				return
			}
			continue
		}

		// The one line that makes -vvv worth turning on here. PullPoint returns its errors
		// instead of swallowing them, so unlike every sdk.Fetch* it logs nothing at all --
		// which left an idle subscription indistinguishable from a stuck one at every
		// verbosity. An empty pull is the normal answer (ONVIF Core section 9.1.2) and this
		// is what says so. Per round and never per record: one line per event would
		// duplicate the whole volume of stdout onto stderr.
		Logger.Trace().Str("addr", dev.Xaddr).Int("messages", len(notifications)).Msg("pulled")

		for _, notification := range notifications {
			if err := out.write(newEventRecord(time.Now(), dev, notification)); err != nil {
				// The writer has already recorded the failure and ended the run; there is
				// nothing this camera can add and nothing it can do about it.
				return
			}
		}

		if wait := pullThrottle(len(notifications), time.Since(started)); wait > 0 {
			if !pause(ctx, wait) {
				return
			}
		}
	}
}

// pullThrottle is how long to wait before pulling this camera again.
//
// Pure over its arguments, like resubscribeDelay and verbosityLevel, so the policy is a table
// in a test rather than a stopwatch. See minPullInterval for why it exists at all.
func pullThrottle(messages int, elapsed time.Duration) time.Duration {
	if messages > 0 || elapsed >= minPullInterval {
		return 0
	}
	return minPullInterval - elapsed
}

// establishAgain retries one camera's subscription, logging the first failure of an outage
// louder than the ones that follow it.
//
// One line per state change rather than one per attempt: a camera that stays unplugged for a
// week would otherwise fill stderr with a warning every few seconds, and an operator reading
// it learns nothing after the first.
func establishAgain(ctx context.Context, dev networking.ClientInfo, open opener, attempt int) stream {
	retry, cancel := context.WithTimeout(ctx, establishTimeout)
	defer cancel()

	subscription, err := open(retry, dev)
	if err == nil {
		return subscription
	}
	if ctx.Err() != nil {
		return nil
	}
	if attempt == 0 {
		Logger.Warn().Str("addr", dev.Xaddr).Err(err).Msg("subscribing again failed")
	} else {
		Logger.Info().Str("addr", dev.Xaddr).Int("attempt", attempt+1).Err(err).
			Msg("subscribing again failed")
	}
	return nil
}

// openPullPoint is the real opener: it connects to the camera with its own credentials and
// creates a pull point on it.
func openPullPoint(ctx context.Context, dev networking.ClientInfo) (stream, error) {
	auth, source := credentialsFor(dev.Uuid)
	cam, err := sdk.NewDevice(ctx, dev, auth, &httpClient)
	if err != nil {
		// Naming the source turns the most confusing failure this tool produces into an
		// actionable one, exactly as dumpSomething does: a 401 answered with the built-in
		// admin/admin looks just like a protocol failure.
		return nil, fmt.Errorf("connecting to %s with the %s credentials: %w", dev.Xaddr, source, err)
	}
	profileS, ok := cam.ProfileS()
	if !ok {
		return nil, ErrNoProfileS
	}
	if !profileS.HasEvent() {
		return nil, ErrNoEventService
	}

	pullPoint := profileS.NewPullPoint(pullTimeout, pullMessageLimit)
	if err := pullPoint.Subscribe(ctx); err != nil {
		return nil, err
	}
	return pullPoint, nil
}

// establishError is the whole exit-status policy of `subscribe`, and it turns on one
// question: can stdout mean anything?
//
// A run that subscribed to nothing prints an empty stream, and an empty stream says the same
// as a quiet fleet -- so it cannot be a success, and it says so at once rather than after the
// operator has waited for events that could never arrive. A run that subscribed to some of
// its targets prints their events, names the rest on stderr, and is a success: over a run
// measured in days the alternative makes the exit status a report on the network weather of
// an arbitrary instant, which nothing can act on. And a run that was stopped is a success by
// construction, which is what this function taking no notion of cancellation says.
//
// Pure over its arguments, like verbosityLevel, so the policy is a table in a test rather
// than a signal sent to a running binary.
func establishError(established, targets int) error {
	if established > 0 {
		return nil
	}
	return fmt.Errorf("no subscription could be established on any of the %d target(s); "+
		"the failures are named on stderr, and `onvif-cli dump event TARGET` says whether a "+
		"camera offers the event service at all", targets)
}

// resubscribeDelay is how long to wait before the attempt-th retry, doubling and capped.
//
// Pure over the attempt count, so the back-off is a table in a test rather than a stopwatch.
// It never returns zero: attempt 1 is already a retry, and something has just failed.
func resubscribeDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := resubscribeDelayMin
	for range attempt - 1 {
		if delay >= resubscribeDelayMax {
			break
		}
		delay *= 2
	}
	return min(delay, resubscribeDelayMax)
}

// pause waits out a back-off and reports whether the wait finished rather than the run
// ending. A time.Sleep would keep a stopped run alive for the whole delay.
func pause(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

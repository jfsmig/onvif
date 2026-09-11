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

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jfsmig/onvif/event"
	"github.com/jfsmig/onvif/xsd"
	"github.com/jfsmig/onvif/xsd/onvif"
)

// ErrNoPullPoint reports a device that accepted the subscription without saying where it
// lives.
//
// Reported rather than shrugged off: the address is the whole result of
// CreatePullPointSubscription, and POSTing to "" would surface later as a URL parse error
// that names nothing and hides the protocol fact.
var ErrNoPullPoint = errors.New("the device returned no pull-point subscription reference")

// terminationMargin multiplies the pull timeout to get the InitialTerminationTime asked of
// the device, and minTermination floors it.
//
// Longer than one pull on purpose. ONVIF Core section 9.1.1 makes PullMessages the keep
// alive -- "after a PullMessages response is returned, the subscription should be active for
// at least the timeout specified in the PullMessages request" -- so this margin only has to
// cover the gap between two pulls, plus whatever it takes to send the next one. That makes
// it a property of the mechanism rather than an operator's choice, which is why it is
// derived here while the timeout itself is the caller's.
//
// The floor is the value ONVIF's own example in section 9.10.3 uses, and it keeps a caller
// who asked for a one-second timeout from asking a device for a subscription shorter than
// the request that creates it.
const (
	terminationMargin = 3
	minTermination    = time.Minute
)

// unsubscribeTimeout bounds the Unsubscribe of a subscription that is being released. It is
// spent on a context of its own; see Unsubscribe.
const unsubscribeTimeout = 5 * time.Second

// PullPoint is one ONVIF real-time pull-point subscription (ONVIF Core section 9.1): the
// client asks the event service for a pull point of its own, then holds it open by pulling
// from it.
//
// This is the client-initiated half of ONVIF eventing and the only half this library
// implements. The push interface would need an inbound listener, and there is none here.
//
// It returns errors, unlike the Fetch* family beside it, and the difference is principled
// rather than an oversight. AGENTS.md has sdk swallow a per-call error so that "one
// unsupported operation cannot lose a whole dump": a Fetch* is a snapshot assembled from
// many independent, mostly optional operations, where a zero field is the truthful answer
// "the camera did not answer this one". A subscription is the opposite on every count -- a
// single chain in which every link is load-bearing, long-lived, and with no partial struct
// to hand back, since without the manager URI there is nothing to pull from. The name says
// which rule applies: this is not a Fetch.
//
// One PullPoint is owned by one goroutine. Subscribe and Unsubscribe both move the manager
// URI that Pull reads, and nothing here is guarded, so sharing one is a data race. One per
// camera, several in parallel, is the intended shape: the networking.Client underneath
// already tolerates concurrent calls, which is what every Fetch* fan-out rests on. This type
// starts no goroutine of its own.
//
// It holds no credential and is never formatted, and nothing here logs an envelope, per the
// secret-hygiene rule in AGENTS.md: the manager URI and the termination time are all that
// reach the log, at debug.
type PullPoint struct {
	profile *ProfileS

	timeout time.Duration
	limit   int32

	// manager is the subscription manager URI, empty before Subscribe and after
	// Unsubscribe. A caller driving the retry loop reads that emptiness as "subscribe
	// before pulling again".
	manager string

	// terminationTime is what the device last told us, kept for the log and for a caller
	// that wants to see it. Nothing here computes with it: PullMessages is the keep alive.
	terminationTime string
}

// NewPullPoint prepares a subscription against this appliance, sending nothing.
//
// timeout is the PullMessages Timeout, so the longest one pull blocks on an idle camera. It
// is the caller's because its real ceiling is the caller's: a pull is one HTTP exchange, so
// a timeout at or above the http.Client Timeout the appliance was built with dies on the
// client side every round and makes a working camera look like one that never answers. Only
// the caller knows that number.
//
// limit caps one reply. ONVIF Core section 9.1.2 says a device shall not fault on a limit
// larger than it supports and shall return up to what it has instead, so this is a request
// and not a contract.
//
// HasEvent reports whether the service is advertised at all, which is worth asking first: a
// camera without it fails here with utils.ErrNoService.
func (p *ProfileS) NewPullPoint(timeout time.Duration, limit int32) *PullPoint {
	return &PullPoint{profile: p, timeout: timeout, limit: limit}
}

// Manager returns the subscription manager URI, empty when not subscribed.
func (pp *PullPoint) Manager() string { return pp.manager }

// TerminationTime returns the expiry the device last reported, empty when not subscribed.
func (pp *PullPoint) TerminationTime() string { return pp.terminationTime }

// Subscribe creates the pull point, replacing any this PullPoint already held.
//
// A pull point it replaces is left to expire at its termination time rather than released.
// Unsubscribe below argues that an orphan is not merely untidy, and that argument holds
// here too -- what bounds it is terminationMargin: an orphan lapses within a minute or so,
// while a caller retrying a flapping camera waits at least a second between attempts, so
// only a couple can exist against event.Capabilities.MaxPullPoints at any time. Releasing
// it instead would mean a round trip to a manager that, in the case this is reached from --
// a pull that failed -- has usually gone already.
//
// No Filter is sent, which is what asks for everything: ONVIF Core section 9.1.1 says "If no
// Filter element is specified the pullpoint shall notify all occurring events to the
// client." No SubscriptionPolicy either -- ONVIF's own reference request in section 9.10.3
// carries none, and tev:ChangedOnly would suppress the Initialized messages that tell a
// subscriber the current state of every property.
func (pp *PullPoint) Subscribe(ctx context.Context) error {
	termination := event.AbsoluteOrRelativeTimeType(secondsDuration(pp.termination()))

	reply, err := pp.profile.CreatePullPointSubscription(ctx, event.CreatePullPointSubscription{
		InitialTerminationTime: &termination,
	})
	if err != nil {
		return fmt.Errorf("CreatePullPointSubscription: %w", err)
	}

	// Trimmed, and that is not cosmetic. The reference response in ONVIF Core section 9.10.4
	// puts the URI on its own indented line inside <wsa:Address>, and encoding/xml hands the
	// newlines through -- so an untrimmed address fails in http.NewRequestWithContext with a
	// URL error that names nothing an operator could act on.
	address := strings.TrimSpace(string(reply.SubscriptionReference.Address))
	if address == "" {
		return ErrNoPullPoint
	}

	pp.manager = pp.profile.client.AtDeviceHost(address)
	pp.terminationTime = strings.TrimSpace(string(reply.TerminationTime))
	Logger.Debug().Str("manager", pp.manager).Str("terminates", pp.terminationTime).
		Msg("pull point created")
	return nil
}

// Pull fetches whatever the device has, blocking until it has something or until the
// timeout NewPullPoint was given expires.
//
// An empty result with no error is the normal idle answer, not a failure: ONVIF Core section
// 9.1.2 makes "no messages waiting, none generated before the Timeout" the third of three
// documented behaviours, and says a device shall not return zero messages any sooner.
//
// The request is addressed to the subscription manager rather than to the event service, and
// that is what event.PullMessages.To expresses; see event/actions.go.
func (pp *PullPoint) Pull(ctx context.Context) ([]Notification, error) {
	if pp.manager == "" {
		return nil, ErrNoPullPoint
	}

	reply, err := pp.profile.PullMessages(ctx, event.PullMessages{
		To:           pp.manager,
		Timeout:      secondsDuration(pp.timeout),
		MessageLimit: xsd.Int(pp.limit),
	})
	if err != nil {
		return nil, fmt.Errorf("PullMessages: %w", err)
	}

	if termination := strings.TrimSpace(string(reply.TerminationTime)); termination != "" {
		pp.terminationTime = termination
	}

	out := make([]Notification, 0, len(reply.NotificationMessage))
	for _, message := range reply.NotificationMessage {
		out = append(out, notifications(message)...)
	}
	return out, nil
}

// Unsubscribe releases the pull point, so the device frees it now rather than at its
// termination time (ONVIF Core sections 9.1.4 and 9.10.7). It is a no-op when there is
// nothing subscribed.
//
// It runs on a context derived with context.WithoutCancel, because by the time this is
// wanted the caller's context is normally already cancelled -- an interrupt is how a
// subscription ends -- and a cancelled context makes the request fail before it leaves the
// process, leaving the device holding the pull point until it expires. That is not merely
// untidy: event.Capabilities.MaxPullPoints is a small number on real devices, so a collector
// a supervisor restarts a few times in a minute would exhaust them and then fail in a way
// that reads exactly like a rejected credential. WithoutCancel keeps the context's values
// and drops only its cancellation; the WithTimeout around it is what stops this becoming a
// way to hang on the way out.
//
// The manager is cleared before the error is examined, so a caller's deferred call cannot
// retry a pull point that has already been told to go away.
func (pp *PullPoint) Unsubscribe(ctx context.Context) error {
	if pp.manager == "" {
		return nil
	}

	to := pp.manager
	pp.manager = ""
	pp.terminationTime = ""

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), unsubscribeTimeout)
	defer cancel()

	if _, err := pp.profile.Unsubscribe(ctx, event.Unsubscribe{To: to}); err != nil {
		return fmt.Errorf("Unsubscribe: %w", err)
	}
	Logger.Debug().Str("manager", to).Msg("pull point released")
	return nil
}

// termination is the InitialTerminationTime to ask for; see terminationMargin.
func (pp *PullPoint) termination() time.Duration {
	return max(pp.timeout*terminationMargin, minTermination)
}

// secondsDuration renders a whole number of seconds as the xs:duration a PullMessages
// Timeout is typed as (ONVIF Core section 9.1.2, docs/wsdl/event.wsdl:129).
//
// Not xsd.Duration.NewDuration: that takes six strings and returns an error this call site
// cannot provoke, since the value comes from a time.Duration this package chose, and
// plumbing an unreachable error through the pull loop would make every caller handle
// something that cannot happen.
func secondsDuration(d time.Duration) xsd.Duration {
	return xsd.Duration("PT" + strconv.FormatInt(int64(d/time.Second), 10) + "S")
}

// Notification is one event, projected onto what a caller prints or forwards.
//
// A projection rather than event.NotificationMessage itself: that holder carries three
// endpoint references and a metadata element a pull point never fills, and the payload a
// reader wants sits four levels down inside it. Handing it out whole would also put event/
// and xsd/onvif/ in the import list of every consumer, which is the opposite of what this
// layer is for.
//
// Every string is trimmed, and that is not cosmetic either: the examples in ONVIF Core
// sections 9.10.4 and 9.10.6 show a device putting the topic and the timestamps on their own
// indented lines, and character data arrives with those newlines in it.
type Notification struct {
	// Topic is the topic expression as the device spelled it, prefixed -- "tns1:..." -- which
	// is the form ONVIF's own documentation uses. The Dialect attribute is not carried: it is
	// the concrete-set dialect in practice, and it describes the grammar rather than the event.
	Topic string

	// UtcTime is when the event occurred, as the device stated it (section 9.4.1). It is not
	// corrected by the clock offset the client learnt: never rewrite what the device said.
	UtcTime string

	// Operation is tt:Message/@PropertyOperation, set only on a property event, where
	// section 9.4.2 gives it Initialized, Changed or Deleted.
	Operation string

	Source []Item
	Key    []Item
	Data   []Item
}

// Item is one tt:SimpleItem or tt:ElementItem of a message group (ONVIF Core section 9.4.1).
//
// A slice of these rather than a map, even though section 9.4.1 requires the names to be
// unique within any group of a message: the device's order is information, and a map here
// would silently merge two items a device gave the same name, which is the kind of thing
// worth seeing rather than smoothing over. Flattening is a rendering decision and belongs
// where the rendering is.
type Item struct {
	Name string
	// Value is a SimpleItem's Value attribute, or an ElementItem's element as raw XML.
	Value string
}

// notifications projects one raw holder, which carries one event per tt:Message it holds.
//
// One holder is not one event: ONVIF Core section 9.4 (page 107) gives wsnt:Message a
// tt:Message with maxOccurs="unbounded", and the topic is the holder's while the time, the
// operation and the three item groups are each message's own. So the topic is repeated across
// the events of one holder, which is what it means, and nothing is merged across them.
func notifications(message event.NotificationMessage) []Notification {
	topic := strings.TrimSpace(string(message.Topic.TopicKinds))

	out := make([]Notification, 0, len(message.Message.Message))
	for _, payload := range message.Message.Message {
		out = append(out, Notification{
			Topic:     topic,
			UtcTime:   strings.TrimSpace(string(payload.UtcTime)),
			Operation: strings.TrimSpace(payload.PropertyOperation),
			Source:    items(payload.Source),
			Key:       items(payload.Key),
			Data:      items(payload.Data),
		})
	}
	return out
}

// items flattens one tt:ItemList, keeping the device's order and putting the simple items
// first, which is the order the schema declares them in.
//
// An ElementItem contributes its element as raw XML. Section 9.4.1 recommends SimpleItem
// "whenever applicable" and says ElementItem should not appear in Source or Key at all, so
// this is the rare case -- but dropping it would lose payload silently, which is worse than
// handing over a fragment the caller has to recognise.
func items(list onvif.ItemList) []Item {
	if len(list.SimpleItem) == 0 && len(list.ElementItem) == 0 {
		return nil
	}

	out := make([]Item, 0, len(list.SimpleItem)+len(list.ElementItem))
	for _, item := range list.SimpleItem {
		out = append(out, Item{
			Name:  strings.TrimSpace(item.Name),
			Value: strings.TrimSpace(string(item.Value)),
		})
	}
	for _, item := range list.ElementItem {
		out = append(out, Item{
			Name:  strings.TrimSpace(item.Name),
			Value: strings.TrimSpace(item.Value),
		})
	}
	return out
}

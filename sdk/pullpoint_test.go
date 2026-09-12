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

// A pull-point subscription could not be completed at all: ONVIF addresses PullMessages and
// Unsubscribe to the subscription manager that CreatePullPointSubscriptionResponse hands
// back -- a wsa:To of that URI, and a POST to it rather than to the event service -- and
// CallMethod routed on the request struct's package name alone.
//
// These tests drive a stub device and assert where each request actually went, which is the
// only thing that proves the mechanism end to end: every unit below it can be right while
// the pull still lands on the wrong endpoint.
//
// Verified against ONVIF Core sections 9.10.3 to 9.10.7 and docs/wsdl/event.wsdl.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jfsmig/onvif/v2/networking"
)

// The subscription manager sits on a path of its own, which is the point: a pull that landed
// on the event service endpoint would still be answered by this one server, so only the path
// distinguishes a working implementation from the broken one.
const (
	eventsPath       = "/onvif/events"
	subscriptionPath = "/onvif/Subscription"
)

// eventStub is a camera that grants pull points and answers pulls, recording the path of
// every request it received.
type eventStub struct {
	srv *httptest.Server

	mu    sync.Mutex
	calls []stubCall

	// subscribeReplies is consumed one per CreatePullPointSubscription, so a test can make
	// the first attempt fail and the second succeed. An exhausted list keeps answering with
	// the last entry.
	subscribeReplies []string
	// pullFailures counts the pulls to answer with a fault before answering normally.
	pullFailures int
	// advertisedHost overrides the host of the subscription URI, standing in for a device
	// that fills it in from its own point of view (section 9.10.4).
	advertisedHost string
	// pullReply, when set, is the body every PullMessages is answered with.
	pullReply string
}

type stubCall struct {
	path string
	body string
}

func newEventStub(t *testing.T) *eventStub {
	t.Helper()

	stub := &eventStub{}
	stub.srv = httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(stub.srv.Close)
	return stub
}

func (s *eventStub) host() string { return strings.TrimPrefix(s.srv.URL, "http://") }

func (s *eventStub) record(r *http.Request, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, stubCall{path: r.URL.Path, body: body})
}

// received reports the paths of every request, in order.
func (s *eventStub) received() []stubCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]stubCall(nil), s.calls...)
}

func (s *eventStub) counted(path string) int {
	n := 0
	for _, call := range s.received() {
		if call.path == path {
			n++
		}
	}
	return n
}

func (s *eventStub) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	request := string(body)
	s.record(r, request)

	w.Header().Set("Content-Type", "application/soap+xml")

	switch {
	case strings.Contains(request, "GetSystemDateAndTime"):
		_, _ = w.Write([]byte(eventSoap(`<tds:GetSystemDateAndTimeResponse/>`)))

	case strings.Contains(request, "GetCapabilities"):
		host := s.host()
		_, _ = w.Write([]byte(eventSoap(
			`<tds:GetCapabilitiesResponse><tds:Capabilities>` +
				`<tt:Device><tt:XAddr>http://` + host + `/onvif/device_service</tt:XAddr></tt:Device>` +
				`<tt:Media><tt:XAddr>http://` + host + `/onvif/media_service</tt:XAddr></tt:Media>` +
				`<tt:Events><tt:XAddr>http://` + host + eventsPath + `</tt:XAddr></tt:Events>` +
				`</tds:Capabilities></tds:GetCapabilitiesResponse>`)))

	case strings.Contains(request, "CreatePullPointSubscription"):
		_, _ = w.Write([]byte(s.nextSubscribeReply()))

	case strings.Contains(request, "PullMessages"):
		if s.pullReply != "" {
			_, _ = w.Write([]byte(eventSoap(s.pullReply)))
			return
		}
		if s.takePullFailure() {
			// What a device that has dropped the pull point actually answers: a fault, which
			// ReadAndParse reports as ErrHTTP plus the status.
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(eventSoap(`<s:Fault/>`)))
			return
		}
		_, _ = w.Write([]byte(eventSoap(pullReply)))

	case strings.Contains(request, "Unsubscribe"):
		_, _ = w.Write([]byte(eventSoap(`<wsnt:UnsubscribeResponse/>`)))

	default:
		_, _ = w.Write([]byte(eventSoap(`<tds:Empty/>`)))
	}
}

func (s *eventStub) nextSubscribeReply() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.subscribeReplies) == 0 {
		host := s.advertisedHost
		if host == "" {
			host = s.host()
		}
		return eventSoap(subscribeReply("http://" + host + subscriptionPath + "?Idx=7"))
	}
	reply := s.subscribeReplies[0]
	if len(s.subscribeReplies) > 1 {
		s.subscribeReplies = s.subscribeReplies[1:]
	}
	return reply
}

func (s *eventStub) takePullFailure() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pullFailures == 0 {
		return false
	}
	s.pullFailures--
	return true
}

func eventSoap(body string) string {
	return `<?xml version="1.0"?><s:Envelope ` +
		`xmlns:s="http://www.w3.org/2003/05/soap-envelope" ` +
		`xmlns:tds="http://www.onvif.org/ver10/device/wsdl" ` +
		`xmlns:tet="http://www.onvif.org/ver10/events/wsdl" ` +
		`xmlns:wsnt="http://docs.oasis-open.org/wsn/b-2" ` +
		`xmlns:wsa="http://www.w3.org/2005/08/addressing" ` +
		`xmlns:tt="http://www.onvif.org/ver10/schema"><s:Body>` + body + `</s:Body></s:Envelope>`
}

// subscribeReply is the shape of ONVIF Core section 9.10.4, indentation included: a device
// really does put the URI on its own line, and an untrimmed address is not a URL.
func subscribeReply(address string) string {
	return `<tet:CreatePullPointSubscriptionResponse>
    <tet:SubscriptionReference>
      <wsa:Address>
        ` + address + `
      </wsa:Address>
    </tet:SubscriptionReference>
    <wsnt:CurrentTime>2026-09-10T14:00:00Z</wsnt:CurrentTime>
    <wsnt:TerminationTime>2026-09-10T14:01:00Z</wsnt:TerminationTime>
  </tet:CreatePullPointSubscriptionResponse>`
}

const pullReply = `<tet:PullMessagesResponse>
  <tet:CurrentTime>2026-09-10T14:00:05Z</tet:CurrentTime>
  <tet:TerminationTime>2026-09-10T14:01:05Z</tet:TerminationTime>
  <wsnt:NotificationMessage>
    <wsnt:Topic Dialect="http://www.onvif.org/ver10/tev/topicExpression/ConcreteSet">
      tns1:RuleEngine/LineDetector/Crossed
    </wsnt:Topic>
    <wsnt:Message>
      <tt:Message UtcTime="2026-09-10T14:00:04.500Z">
        <tt:Source><tt:SimpleItem Name="Rule" Value="MyImportantFence1"/></tt:Source>
        <tt:Data><tt:SimpleItem Name="ObjectId" Value="15"/></tt:Data>
      </tt:Message>
    </wsnt:Message>
  </wsnt:NotificationMessage>
</tet:PullMessagesResponse>`

// batchedPullReply is one holder carrying two tt:Message, which ONVIF Core section 9.4 (page
// 107) allows with maxOccurs="unbounded". The topic belongs to the holder and the times and
// data belong to each message.
const batchedPullReply = `<tet:PullMessagesResponse>
  <tet:CurrentTime>2026-09-10T14:00:05Z</tet:CurrentTime>
  <tet:TerminationTime>2026-09-10T14:01:05Z</tet:TerminationTime>
  <wsnt:NotificationMessage>
    <wsnt:Topic>tns1:RuleEngine/LineDetector/Crossed</wsnt:Topic>
    <wsnt:Message>
      <tt:Message UtcTime="2026-09-10T14:00:04.500Z">
        <tt:Data><tt:SimpleItem Name="ObjectId" Value="15"/></tt:Data>
      </tt:Message>
      <tt:Message UtcTime="2026-09-10T14:00:04.900Z">
        <tt:Data><tt:SimpleItem Name="ObjectId" Value="16"/></tt:Data>
      </tt:Message>
    </wsnt:Message>
  </wsnt:NotificationMessage>
</tet:PullMessagesResponse>`

// subscribed connects to the stub and creates a pull point on it.
func subscribed(t *testing.T, stub *eventStub) *PullPoint {
	t.Helper()

	appliance, err := NewDevice(context.Background(),
		networking.ClientInfo{Xaddr: stub.host()},
		networking.ClientAuth{Username: "admin", Password: "admin"},
		stub.srv.Client())
	if err != nil {
		t.Fatalf("NewDevice: %v", err)
	}
	profileS, ok := appliance.ProfileS()
	if !ok {
		t.Fatal("no Profile S client")
	}
	if !profileS.HasEvent() {
		t.Fatal("the stub advertises an Events endpoint but HasEvent says otherwise")
	}

	pullPoint := profileS.NewPullPoint(2*time.Second, 10)
	if err := pullPoint.Subscribe(context.Background()); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	return pullPoint
}

// The assertion the whole feature rests on: the pull goes to the subscription manager and
// carries a wsa:To of it, not to the event service endpoint that CallMethod's package-name
// routing would have chosen.
func TestPullMessagesGoesToTheSubscriptionManager(t *testing.T) {
	stub := newEventStub(t)
	pullPoint := subscribed(t, stub)

	if got := pullPoint.Manager(); !strings.HasSuffix(got, subscriptionPath+"?Idx=7") {
		t.Fatalf("Manager() = %q, want the subscription URI — the reply's address did not "+
			"bind, or the indentation of section 9.10.4 was not trimmed off it", got)
	}

	notifications, err := pullPoint.Pull(context.Background())
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}

	// The subscription was requested at the event service, and pulled from the manager.
	if got := stub.counted(eventsPath); got != 1 {
		t.Errorf("the event service endpoint received %d requests, want just the "+
			"CreatePullPointSubscription", got)
	}
	if got := stub.counted(subscriptionPath); got != 1 {
		t.Fatalf("the subscription manager received %d requests, want the PullMessages — a "+
			"pull addressed to the event service reaches no subscription", got)
	}

	var pull string
	for _, call := range stub.received() {
		if call.path == subscriptionPath {
			pull = call.body
		}
	}
	if !strings.Contains(pull, "<wsa:To>") {
		t.Errorf("the pull carries no wsa:To, which ONVIF Core section 9.10.5 shows on it: %s", pull)
	}
	if !strings.Contains(pull, subscriptionPath) {
		t.Errorf("the wsa:To does not name the subscription manager: %s", pull)
	}

	// And the payload made it back out, which is the other half of a subscription being
	// usable at all.
	if len(notifications) != 1 {
		t.Fatalf("got %d notifications, want 1", len(notifications))
	}
	if got := notifications[0].Topic; got != "tns1:RuleEngine/LineDetector/Crossed" {
		t.Errorf("Topic = %q, want it trimmed of the indentation the device sent", got)
	}
	if got := notifications[0].UtcTime; got != "2026-09-10T14:00:04.500Z" {
		t.Errorf("UtcTime = %q", got)
	}
	if len(notifications[0].Data) != 1 || notifications[0].Data[0].Name != "ObjectId" {
		t.Errorf("Data = %+v, want the ObjectId item", notifications[0].Data)
	}
}

// A device fills the subscription URI in from its own point of view, which ONVIF Core
// section 9.10.4 shows: its example answers 160.10.64.10, an address that need not resolve
// from the client. AtDeviceHost re-points it, keeping the port the device chose.
func TestAnAdvertisedHostIsRePointedAtTheDevice(t *testing.T) {
	stub := newEventStub(t)
	stub.advertisedHost = "160.10.64.10:" + strings.Split(stub.host(), ":")[1]

	pullPoint := subscribed(t, stub)
	if _, err := pullPoint.Pull(context.Background()); err != nil {
		t.Fatalf("Pull: %v — the advertised host was not re-pointed, so the pull went "+
			"somewhere unreachable", err)
	}
	if got := stub.counted(subscriptionPath); got != 1 {
		t.Errorf("the subscription manager received %d pulls, want 1", got)
	}
}

// A device that accepts the subscription without saying where it lives has granted nothing
// usable, and POSTing to "" would surface later as a URL parse error naming nothing.
func TestSubscribeWithoutAnAddressIsReported(t *testing.T) {
	stub := newEventStub(t)
	stub.subscribeReplies = []string{eventSoap(
		`<tet:CreatePullPointSubscriptionResponse>` +
			`<tet:SubscriptionReference></tet:SubscriptionReference>` +
			`</tet:CreatePullPointSubscriptionResponse>`)}

	appliance, err := NewDevice(context.Background(),
		networking.ClientInfo{Xaddr: stub.host()}, networking.ClientAuth{}, stub.srv.Client())
	if err != nil {
		t.Fatalf("NewDevice: %v", err)
	}
	profileS, _ := appliance.ProfileS()

	err = profileS.NewPullPoint(2*time.Second, 10).Subscribe(context.Background())
	if !errors.Is(err, ErrNoPullPoint) {
		t.Errorf("Subscribe = %v, want ErrNoPullPoint", err)
	}
}

// Unsubscribe runs on a context detached from the caller's, because by the time a
// subscription is released the caller's is normally already cancelled -- an interrupt is how
// a subscription ends. Without the detachment the request fails before it leaves the
// process, and a device with a small MaxPullPoints then refuses the next subscription until
// the old one expires.
//
// This is the only way to observe that: the request arriving after the cancellation is the
// property, and nothing else distinguishes it.
func TestUnsubscribeSurvivesACancelledContext(t *testing.T) {
	stub := newEventStub(t)
	pullPoint := subscribed(t, stub)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := pullPoint.Unsubscribe(ctx); err != nil {
		t.Fatalf("Unsubscribe: %v — a cancelled run cannot release its pull point, so the "+
			"device holds it until its termination time", err)
	}
	if got := stub.counted(subscriptionPath); got != 1 {
		t.Errorf("the subscription manager received %d requests, want the Unsubscribe", got)
	}
	if pullPoint.Manager() != "" {
		t.Error("the manager URI survived Unsubscribe, so a deferred second call would retry it")
	}
}

// A pull against a released subscription is a mistake worth naming rather than a POST to "".
func TestPullWithoutASubscriptionIsReported(t *testing.T) {
	stub := newEventStub(t)
	pullPoint := subscribed(t, stub)

	if err := pullPoint.Unsubscribe(context.Background()); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}
	if _, err := pullPoint.Pull(context.Background()); !errors.Is(err, ErrNoPullPoint) {
		t.Errorf("Pull after Unsubscribe = %v, want ErrNoPullPoint", err)
	}
}

// ONVIF Core section 9.1.1 makes PullMessages the keep-alive, so the subscription only has
// to outlive the gap between two pulls -- but it does have to outlive it, and asking a
// device for a termination shorter than the request that refreshes it would be a subscription
// that expires while the client is waiting on it.
func TestTheTerminationTimeAskedForOutlivesAPull(t *testing.T) {
	for _, timeout := range []time.Duration{time.Second, 10 * time.Second, time.Minute} {
		pullPoint := &PullPoint{timeout: timeout}
		if got := pullPoint.termination(); got <= timeout {
			t.Errorf("a %s pull asks for a %s subscription, which can expire mid-pull",
				timeout, got)
		}
	}
}

// Section 9.1.2 types the Timeout as xs:duration, and xsd.Duration's own constructor renders
// a seconds-only duration as "PT0S" -- a zero timeout. This is the rendering that goes on the
// wire instead.
func TestSecondsDuration(t *testing.T) {
	for _, tc := range []struct {
		in   time.Duration
		want string
	}{
		{10 * time.Second, "PT10S"},
		{time.Minute, "PT60S"},
		{90 * time.Second, "PT90S"},
	} {
		if got := string(secondsDuration(tc.in)); got != tc.want {
			t.Errorf("secondsDuration(%s) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// One holder is not one event. A single onvif.Message in the holder merged the messages it
// held rather than dropping them -- the attributes came from the last and the item slices
// appended across all of them -- so a subscriber received one event stamped with the second
// one's time carrying both one's data. ONVIF Core section 9.4 (page 107) gives the holder
// maxOccurs="unbounded", and the projection has to open it out.
func TestABatchedHolderBecomesOneNotificationPerMessage(t *testing.T) {
	stub := newEventStub(t)
	stub.pullReply = batchedPullReply
	pullPoint := subscribed(t, stub)

	got, err := pullPoint.Pull(context.Background())
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("one holder with two messages produced %d notifications, want 2", len(got))
	}
	for i, want := range []struct{ utc, object string }{
		{"2026-09-10T14:00:04.500Z", "15"},
		{"2026-09-10T14:00:04.900Z", "16"},
	} {
		if got[i].UtcTime != want.utc {
			t.Errorf("notification %d UtcTime = %q, want %q", i, got[i].UtcTime, want.utc)
		}
		if len(got[i].Data) != 1 {
			t.Fatalf("notification %d carries %d data items, want 1 — the two events were "+
				"merged", i, len(got[i].Data))
		}
		if got[i].Data[0].Value != want.object {
			t.Errorf("notification %d ObjectId = %q, want %q", i, got[i].Data[0].Value, want.object)
		}
		// The topic is the holder's, so both events carry it.
		if got[i].Topic != "tns1:RuleEngine/LineDetector/Crossed" {
			t.Errorf("notification %d Topic = %q", i, got[i].Topic)
		}
	}
}

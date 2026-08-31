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

package event

import (
	"encoding/xml"
	"strings"
	"testing"
)

// Struct tags of the form `xml:"wsnt:Address"` do not declare a namespace: Go's separator
// is a space, so the whole string becomes the element name and an incoming <wsa:Address>
// never matches. The tags happen to serialise correctly, which is why requests work and
// responses silently come back empty.
//
// The case that matters is the pull-point address. Without it there is no event flow at all.

// pullPointResponse is the shape a real camera returns, prefixes and all.
const pullPointResponse = `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
            xmlns:tev="http://www.onvif.org/ver10/events/wsdl"
            xmlns:wsnt="http://docs.oasis-open.org/wsn/b-2"
            xmlns:wsa="http://www.w3.org/2005/08/addressing">
 <s:Body>
  <tev:CreatePullPointSubscriptionResponse>
    <tev:SubscriptionReference>
      <wsa:Address>http://192.168.1.70:8000/onvif/Subscription?Idx=7</wsa:Address>
    </tev:SubscriptionReference>
    <wsnt:CurrentTime>2026-08-31T10:00:00Z</wsnt:CurrentTime>
  </tev:CreatePullPointSubscriptionResponse>
 </s:Body>
</s:Envelope>`

const wantPullPoint = "http://192.168.1.70:8000/onvif/Subscription?Idx=7"

func TestPullPointAddressUnmarshals(t *testing.T) {
	// The envelope shape the generated Call_CreatePullPointSubscription unmarshals into.
	type Envelope struct {
		Header struct{}
		Body   struct {
			CreatePullPointSubscriptionResponse CreatePullPointSubscriptionResponse
		}
	}

	var e Envelope
	if err := xml.Unmarshal([]byte(pullPointResponse), &e); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	got := string(e.Body.CreatePullPointSubscriptionResponse.SubscriptionReference.Address)
	if got != wantPullPoint {
		t.Fatalf("SubscriptionReference.Address = %q, want %q — without it there is no pull point "+
			"and the whole event subscription flow is dead", got, wantPullPoint)
	}
}

// The wsa namespace must survive on the request side too: ConsumerReference is marshalled
// into Subscribe, and WS-Addressing requires Address to be in the addressing namespace.
func TestEndpointReferenceMarshalsInTheAddressingNamespace(t *testing.T) {
	body, err := xml.Marshal(EndpointReferenceType{Address: "http://listener/notify"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(body), "http://www.w3.org/2005/08/addressing") {
		t.Fatalf("marshalled EndpointReferenceType does not carry the WS-Addressing namespace: %s", body)
	}
	if !strings.Contains(string(body), "http://listener/notify") {
		t.Fatalf("marshalled EndpointReferenceType lost the address: %s", body)
	}
}

// FilterType's children really are wsnt, unlike EndpointReferenceType's; check the
// conversion kept them there rather than blanket-rewriting every prefix.
func TestFilterUnmarshalsFromTheNotificationNamespace(t *testing.T) {
	const doc = `<Filter xmlns:wsnt="http://docs.oasis-open.org/wsn/b-2">
	  <wsnt:TopicExpression Dialect="http://www.onvif.org/ver10/tev/topicExpression/ConcreteSet">tns1:VideoSource//.</wsnt:TopicExpression>
	</Filter>`

	var f FilterType
	if err := xml.Unmarshal([]byte(doc), &f); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := string(f.TopicExpression.TopicKinds); !strings.Contains(got, "VideoSource") {
		t.Fatalf("TopicExpression.TopicKinds = %q, want it to carry the topic", got)
	}
}

// --- event/operation.go ---
//
// Verified against the authoritative OASIS b-2 schema (targetNamespace
// http://docs.oasis-open.org/wsn/b-2, elementFormDefault="qualified") and against
// docs/wsdl/event.wsdl, rather than from memory.

const subscribeResponse = `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
            xmlns:wsnt="http://docs.oasis-open.org/wsn/b-2"
            xmlns:wsa="http://www.w3.org/2005/08/addressing">
 <s:Body><wsnt:SubscribeResponse>
   <wsnt:SubscriptionReference><wsa:Address>http://cam/onvif/Subscription?Idx=3</wsa:Address></wsnt:SubscriptionReference>
   <wsnt:CurrentTime>2026-08-31T10:00:00Z</wsnt:CurrentTime>
   <wsnt:TerminationTime>2026-08-31T10:05:00Z</wsnt:TerminationTime>
 </wsnt:SubscribeResponse></s:Body></s:Envelope>`

func TestSubscribeResponseUnmarshals(t *testing.T) {
	type Envelope struct {
		Header struct{}
		Body   struct{ SubscribeResponse SubscribeResponse }
	}
	var e Envelope
	if err := xml.Unmarshal([]byte(subscribeResponse), &e); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	r := e.Body.SubscribeResponse
	// b-2.xsd names this local element SubscriptionReference; the field used to be called
	// ConsumerReference, which is the Subscribe *request*'s field.
	if got := string(r.SubscriptionReference.Address); got != "http://cam/onvif/Subscription?Idx=3" {
		t.Fatalf("SubscriptionReference.Address = %q, want the subscription URL", got)
	}
	if got := string(r.CurrentTime); !strings.Contains(got, "2026-08-31") {
		t.Fatalf("CurrentTime = %q", got)
	}
	if got := string(r.TerminationTime); !strings.Contains(got, "10:05") {
		t.Fatalf("TerminationTime = %q", got)
	}
}

func TestRenewResponseUnmarshals(t *testing.T) {
	const doc = `<RenewResponse xmlns:wsnt="http://docs.oasis-open.org/wsn/b-2">
	  <wsnt:TerminationTime>2026-08-31T11:00:00Z</wsnt:TerminationTime>
	  <wsnt:CurrentTime>2026-08-31T10:00:00Z</wsnt:CurrentTime>
	</RenewResponse>`
	var r RenewResponse
	if err := xml.Unmarshal([]byte(doc), &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := string(r.TerminationTime); !strings.Contains(got, "11:00") {
		t.Fatalf("TerminationTime = %q, want the renewed expiry", got)
	}
	if got := string(r.CurrentTime); !strings.Contains(got, "10:00") {
		t.Fatalf("CurrentTime = %q", got)
	}
}

// event.wsdl declares both of these as local elements of the tev schema, so they belong to
// tev — and the policy element name had a stray leading 's', so a camera silently ignored it.
func TestCreatePullPointRequestElementNames(t *testing.T) {
	b, err := xml.Marshal(CreatePullPointSubscription{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, "sSubscriptionPolicy") {
		t.Fatalf("the sSubscriptionPolicy typo is back: %s", got)
	}
	for _, want := range []string{
		`<SubscriptionPolicy xmlns="http://www.onvif.org/ver10/events/wsdl"`,
		`<InitialTerminationTime xmlns="http://www.onvif.org/ver10/events/wsdl"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in: %s", want, got)
		}
	}
}

// The remaining 17 tags are request-only and verified correct; pin their wire format so a
// future blanket rewrite cannot change them silently.
func TestSubscribeRequestWireFormatUnchanged(t *testing.T) {
	b, err := xml.Marshal(Subscribe{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, want := range []string{"<wsnt:Subscribe>", "<wsnt:ConsumerReference>", "<wsnt:Filter>"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("Subscribe wire format changed, missing %s: %s", want, b)
		}
	}
}

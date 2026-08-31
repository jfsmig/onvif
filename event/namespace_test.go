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

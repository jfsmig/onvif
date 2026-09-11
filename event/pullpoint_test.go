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

// Three separate slips stood between a pull point and its events, and this file pins all
// three against one fixture: the PullMessagesResponse of ONVIF Core section 9.10.6, copied
// with its own indentation.
//
//   - PullMessagesResponse.NotificationMessage was a single value, where
//     docs/wsdl/event.wsdl:156 declares minOccurs="0" maxOccurs="unbounded" and section
//     9.1.2 Table 81 says [0][unbounded]. encoding/xml assigns each match to the same field
//     in turn, so only the last of a batch survived -- and a MessageLimit above one makes a
//     batch the normal case.
//   - The payload was typed Message = xsd.AnyType, that is a string, so what bound was the
//     character data of <wsnt:Message>: whitespace, since its content is an element. Every
//     notification arrived with its topic and without its event.
//   - ItemList held one SimpleItem, where section 9.4.1 gives each group "an arbitrary
//     number of items" and the fixture below puts three in one tt:Source.
//
// The indentation is not decoration either. The examples in sections 9.10.4 and 9.10.6 put
// the topic and the timestamps on their own indented lines, and encoding/xml hands those
// newlines through -- which is why sdk trims every value it takes from character data, and
// why TestCharacterDataArrivesIndented exists to keep that from looking like superstition.

import (
	"encoding/xml"
	"strings"
	"testing"
)

// pullMessagesResponse is the reference reply of ONVIF Core section 9.10.6: two objects
// crossing the lines of rules MyImportantFence1 and MyImportantFence2.
const pullMessagesResponse = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope
  xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope"
  xmlns:wsnt="http://docs.oasis-open.org/wsn/b-2"
  xmlns:tet="http://www.onvif.org/ver10/events/wsdl"
  xmlns:tns1="http://www.onvif.org/ver10/topics"
  xmlns:tt="http://www.onvif.org/ver10/schema">
  <SOAP-ENV:Body>
    <tet:PullMessagesResponse>
      <tet:CurrentTime>
        2008-10-10T12:24:58
      </tet:CurrentTime>
      <tet:TerminationTime>
        2008-10-10T12:25:58
      </tet:TerminationTime>
      <wsnt:NotificationMessage>
        <wsnt:Topic Dialect="http://www.onvif.org/ver10/tev/topicExpression/ConcreteSet">
          tns1:RuleEngine/LineDetector/Crossed
        </wsnt:Topic>
        <wsnt:Message>
          <tt:Message UtcTime="2008-10-10T12:24:57.321Z">
            <tt:Source>
              <tt:SimpleItem Name="VideoSourceConfigurationToken" Value="1"/>
              <tt:SimpleItem Name="VideoAnalyticsConfigurationToken" Value="2"/>
              <tt:SimpleItem Value="MyImportantFence1" Name="Rule"/>
            </tt:Source>
            <tt:Data>
              <tt:SimpleItem Name="ObjectId" Value="15"/>
            </tt:Data>
          </tt:Message>
        </wsnt:Message>
      </wsnt:NotificationMessage>
      <wsnt:NotificationMessage>
        <wsnt:Topic Dialect="http://www.onvif.org/ver10/tev/topicExpression/ConcreteSet">
          tns1:RuleEngine/LineDetector/Crossed
        </wsnt:Topic>
        <wsnt:Message>
          <tt:Message UtcTime="2008-10-10T12:24:57.789Z">
            <tt:Source>
              <tt:SimpleItem Name="VideoSourceConfigurationToken" Value="1"/>
              <tt:SimpleItem Name="VideoAnalyticsConfigurationToken" Value="2"/>
              <tt:SimpleItem Value="MyImportantFence2" Name="Rule"/>
            </tt:Source>
            <tt:Data>
              <tt:SimpleItem Name="ObjectId" Value="19"/>
            </tt:Data>
          </tt:Message>
        </wsnt:Message>
      </wsnt:NotificationMessage>
    </tet:PullMessagesResponse>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

// pulled unmarshals the fixture through the same envelope shape the generated
// Call_PullMessages uses.
func pulled(t *testing.T) PullMessagesResponse {
	t.Helper()

	type Envelope struct {
		Header struct{}
		Body   struct {
			PullMessagesResponse PullMessagesResponse
		}
	}
	var e Envelope
	if err := xml.Unmarshal([]byte(pullMessagesResponse), &e); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return e.Body.PullMessagesResponse
}

func TestPullMessagesResponseKeepsEveryNotification(t *testing.T) {
	reply := pulled(t)

	if got := len(reply.NotificationMessage); got != 2 {
		t.Fatalf("got %d notifications, want 2 — a batched reply loses every message but "+
			"one, which is the normal case for any MessageLimit above 1", got)
	}
	for i, want := range []string{"MyImportantFence1", "MyImportantFence2"} {
		source := reply.NotificationMessage[i].Message.Message[0].Source.SimpleItem
		found := false
		for _, item := range source {
			if item.Name == "Rule" && string(item.Value) == want {
				found = true
			}
		}
		if !found {
			t.Errorf("notification %d does not carry Rule=%s; the two messages of the "+
				"reply are not distinct", i, want)
		}
	}
}

func TestNotificationCarriesItsMessagePayload(t *testing.T) {
	reply := pulled(t)
	first := reply.NotificationMessage[0]

	if got := strings.TrimSpace(string(first.Topic.TopicKinds)); got != "tns1:RuleEngine/LineDetector/Crossed" {
		t.Errorf("Topic = %q, want the concrete topic expression", got)
	}

	if got := len(first.Message.Message); got != 1 {
		t.Fatalf("the holder carries %d messages, want 1", got)
	}
	payload := first.Message.Message[0]
	if got := string(payload.UtcTime); got != "2008-10-10T12:24:57.321Z" {
		t.Fatalf("Message.UtcTime = %q — the tt:Message payload does not bind at all, so "+
			"every notification arrives without its event", got)
	}

	// Three items in one group: a single-valued ItemList kept the last and dropped the
	// source identity of the event.
	if got := len(payload.Source.SimpleItem); got != 3 {
		t.Fatalf("Source has %d items, want the 3 of section 9.10.6", got)
	}
	want := map[string]string{
		"VideoSourceConfigurationToken":    "1",
		"VideoAnalyticsConfigurationToken": "2",
		"Rule":                             "MyImportantFence1",
	}
	for _, item := range payload.Source.SimpleItem {
		if expected, known := want[item.Name]; !known {
			t.Errorf("unexpected Source item %q", item.Name)
		} else if got := string(item.Value); got != expected {
			t.Errorf("Source[%s] = %q, want %q", item.Name, got, expected)
		}
	}

	if got := len(payload.Data.SimpleItem); got != 1 {
		t.Fatalf("Data has %d items, want 1", got)
	}
	if got := string(payload.Data.SimpleItem[0].Value); got != "15" {
		t.Errorf("Data[ObjectId] = %q, want 15", got)
	}

	// Not a property event, so the attribute is absent rather than blank.
	if payload.PropertyOperation != "" {
		t.Errorf("PropertyOperation = %q on an ordinary event", payload.PropertyOperation)
	}
}

// The values a device sends as character data arrive with the indentation the specification's
// own examples show, so trimming them is necessary and not cosmetic. Asserted here so that
// the trims in sdk cannot be tidied away as superstition: an untrimmed subscription address
// fails in http.NewRequestWithContext with a URL error that names nothing.
func TestCharacterDataArrivesIndented(t *testing.T) {
	reply := pulled(t)

	if raw := string(reply.CurrentTime); raw == strings.TrimSpace(raw) {
		t.Errorf("CurrentTime = %q arrived already trimmed; the fixture no longer matches "+
			"the indentation of ONVIF Core section 9.10.6 and this stops proving anything", raw)
	}
	if raw := string(reply.NotificationMessage[0].Topic.TopicKinds); raw == strings.TrimSpace(raw) {
		t.Errorf("Topic = %q arrived already trimmed, same problem", raw)
	}
}

// A property event, which is the only place PropertyOperation appears (ONVIF Core section
// 9.4.2): the operation mode is what tells a subscriber that the forty records arriving at
// subscription time are current state and not transitions.
func TestPropertyOperationBinds(t *testing.T) {
	const doc = `<wsnt:NotificationMessage
  xmlns:wsnt="http://docs.oasis-open.org/wsn/b-2"
  xmlns:tt="http://www.onvif.org/ver10/schema">
  <wsnt:Topic>tns1:VideoSource/MotionAlarm</wsnt:Topic>
  <wsnt:Message>
    <tt:Message UtcTime="2026-09-10T14:02:40Z" PropertyOperation="Initialized">
      <tt:Source><tt:SimpleItem Name="VideoSourceConfigurationToken" Value="1"/></tt:Source>
      <tt:Data><tt:SimpleItem Name="State" Value="true"/></tt:Data>
    </tt:Message>
  </wsnt:Message>
</wsnt:NotificationMessage>`

	var message NotificationMessage
	if err := xml.Unmarshal([]byte(doc), &message); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := message.Message.Message[0].PropertyOperation; got != "Initialized" {
		t.Errorf("PropertyOperation = %q, want Initialized", got)
	}
}

// The zero request is the one that asks for everything, and it could not be sent: every
// child was a value, encoding/xml's omitempty ignores struct kinds, so all three went out
// empty -- including a wsnt:TopicExpression with an empty Dialect, which is a topic
// expression in no dialect and which b-2 answers with a fault.
//
// ONVIF Core section 9.1.1: "If no Filter element is specified the pullpoint shall notify all
// occurring events to the client."
func TestCreatePullPointSubscriptionOmitsWhatIsNotSet(t *testing.T) {
	b, err := xml.Marshal(CreatePullPointSubscription{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)

	for _, unwanted := range []string{"Filter", "SubscriptionPolicy", "InitialTerminationTime"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("the subscribe-to-everything request still carries %s: %s", unwanted, got)
		}
	}
}

// AbsoluteOrRelativeTimeType embedded xsd.DateTime and xsd.Duration, and encoding/xml does
// not flatten an anonymous field whose type is not a struct -- it names the element after the
// field, which for an embedded one is its type. So an InitialTerminationTime went out
// carrying <DateTime/> and <Duration/>, where b-2's
// <xsd:union memberTypes="xsd:dateTime xsd:duration"/> admits no children at all. That was
// the BUG(r) marker on CreatePullPointSubscription.
func TestInitialTerminationTimeIsOneSimpleValue(t *testing.T) {
	termination := AbsoluteOrRelativeTimeType("PT1M")
	b, err := xml.Marshal(CreatePullPointSubscription{InitialTerminationTime: &termination})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)

	if !strings.Contains(got, ">PT1M<") {
		t.Errorf("the termination time is not the element's content: %s", got)
	}
	for _, unwanted := range []string{"DateTime", "Duration"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("a union still marshals a %s child element: %s", unwanted, got)
		}
	}
}

// A filter on a topic alone must not drag an empty message-content filter along with it, for
// the same reason: an expression with no Dialect is not one any dialect accepts.
func TestATopicFilterCarriesNoEmptyMessageContent(t *testing.T) {
	b, err := xml.Marshal(CreatePullPointSubscription{
		Filter: &FilterType{
			TopicExpression: &TopicExpressionType{
				Dialect:    "http://www.onvif.org/ver10/tev/topicExpression/ConcreteSet",
				TopicKinds: "tns1:RuleEngine//.",
			},
		},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)

	if !strings.Contains(got, "tns1:RuleEngine//.") {
		t.Errorf("the topic expression did not survive: %s", got)
	}
	if strings.Contains(got, "MessageContent") {
		t.Errorf("an empty message-content filter went along with the topic: %s", got)
	}
}

// A wsnt:Message holds one OR MORE tt:Message: ONVIF Core section 9.4 (page 107) restates
// b-2's NotificationMessageHolderType with
//
//	<tt:element name="Message" type="tt:Message" maxOccurs="unbounded"/>
//
// and says in prose that the holder carries "one or more notification messages of type
// tt:Message". b-2.xsd is not vendored in this tree, so section 9.4 is the citation.
//
// MessageHolder.Message was a single onvif.Message, and that did not drop the extra messages,
// it merged them: encoding/xml assigned each tt:Message to the same field in turn, so UtcTime
// and PropertyOperation came from the LAST one while Source, Key and Data -- slices as of the
// same change that added this file -- appended across all of them. One record went out
// stamped with the second event's time carrying both events' data items, which is an event
// that never occurred rather than an event that went missing. Making ItemList plural is what
// turned the drop into the merge, so this test is the other half of that fix.
func TestEveryMessageInOneHolderIsKept(t *testing.T) {
	const batched = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope
  xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope"
  xmlns:wsnt="http://docs.oasis-open.org/wsn/b-2"
  xmlns:tet="http://www.onvif.org/ver10/events/wsdl"
  xmlns:tt="http://www.onvif.org/ver10/schema">
  <SOAP-ENV:Body>
    <tet:PullMessagesResponse>
      <tet:CurrentTime>2008-10-10T12:24:58Z</tet:CurrentTime>
      <tet:TerminationTime>2008-10-10T12:25:58Z</tet:TerminationTime>
      <wsnt:NotificationMessage>
        <wsnt:Topic>tns1:RuleEngine/LineDetector/Crossed</wsnt:Topic>
        <wsnt:Message>
          <tt:Message UtcTime="2008-10-10T12:24:57.321Z">
            <tt:Data><tt:SimpleItem Name="ObjectId" Value="15"/></tt:Data>
          </tt:Message>
          <tt:Message UtcTime="2008-10-10T12:24:57.999Z">
            <tt:Data><tt:SimpleItem Name="ObjectId" Value="16"/></tt:Data>
          </tt:Message>
        </wsnt:Message>
      </wsnt:NotificationMessage>
    </tet:PullMessagesResponse>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

	type Envelope struct {
		Header struct{}
		Body   struct {
			PullMessagesResponse PullMessagesResponse
		}
	}
	var e Envelope
	if err := xml.Unmarshal([]byte(batched), &e); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	held := e.Body.PullMessagesResponse.NotificationMessage[0].Message.Message
	if got := len(held); got != 2 {
		t.Fatalf("the holder kept %d messages, want the 2 that section 9.4's "+
			"maxOccurs=unbounded allows", got)
	}
	for i, want := range []struct{ utc, object string }{
		{"2008-10-10T12:24:57.321Z", "15"},
		{"2008-10-10T12:24:57.999Z", "16"},
	} {
		if got := string(held[i].UtcTime); got != want.utc {
			t.Errorf("message %d UtcTime = %q, want %q", i, got, want.utc)
		}
		// One data item each, and not both events' items in one list: a merged holder is an
		// event the device never sent, which is worse than one it sent and we lost.
		if got := len(held[i].Data.SimpleItem); got != 1 {
			t.Fatalf("message %d carries %d data items, want 1 — the two events were merged "+
				"into one", i, got)
		}
		if got := string(held[i].Data.SimpleItem[0].Value); got != want.object {
			t.Errorf("message %d ObjectId = %q, want %q", i, got, want.object)
		}
	}
}

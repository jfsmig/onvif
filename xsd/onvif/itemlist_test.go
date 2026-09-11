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

package onvif

// ItemList held one SimpleItem and one ElementItem, where ONVIF Core section 9.4.1 says each
// group "can hold an arbitrary number of items of type SimpleItem or ElementItem" and the
// notification in section 9.10.6 puts three tt:SimpleItem in one tt:Source. encoding/xml
// assigns each match to the same field in turn, so only the last survived: in a notification
// the source identity of the event collapsed to its last token, and in an analytics dump
// every module carrying more than one parameter reported one.
//
// ElementItem modelled only its name, so its value -- one XML element, which for a rule is
// the polygon -- was discarded outright, encoding/xml dropping what no field claims.
//
// Verified against ONVIF Core sections 9.4.1 and 9.10.6. tt:Message is pinned here too
// rather than in event/, because that is where the type lives: the element is in
// http://www.onvif.org/ver10/schema, and event/ holds only the wsnt holder that quotes it.

import (
	"encoding/xml"
	"strings"
	"testing"
)

// The tt:Source of ONVIF Core section 9.10.6, three items in one group.
const messageSource = `<Source xmlns="http://www.onvif.org/ver10/schema">
  <SimpleItem Name="VideoSourceConfigurationToken" Value="1"/>
  <SimpleItem Name="VideoAnalyticsConfigurationToken" Value="2"/>
  <SimpleItem Value="MyImportantFence1" Name="Rule"/>
</Source>`

func TestItemListHoldsEveryItem(t *testing.T) {
	var list ItemList
	if err := xml.Unmarshal([]byte(messageSource), &list); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got := len(list.SimpleItem); got != 3 {
		t.Fatalf("ItemList kept %d of the 3 items in section 9.10.6's tt:Source", got)
	}
	// The device's order, which is why these are a slice: an item list is not a set, and a
	// caller that renders it as one should be the one deciding to.
	for i, want := range []string{
		"VideoSourceConfigurationToken", "VideoAnalyticsConfigurationToken", "Rule",
	} {
		if got := list.SimpleItem[i].Name; got != want {
			t.Errorf("SimpleItem[%d].Name = %q, want %q — the device's order is not kept", i, got, want)
		}
	}
}

func TestElementItemKeepsItsValue(t *testing.T) {
	const doc = `<Parameters xmlns="http://www.onvif.org/ver10/schema">
  <ElementItem Name="Field"><PolygonConfiguration>17</PolygonConfiguration></ElementItem>
</Parameters>`

	var list ItemList
	if err := xml.Unmarshal([]byte(doc), &list); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := len(list.ElementItem); got != 1 {
		t.Fatalf("ItemList kept %d element items, want 1", got)
	}
	if got := list.ElementItem[0].Name; got != "Field" {
		t.Errorf("ElementItem.Name = %q, want Field", got)
	}
	// Section 9.4.1: "In the case of an ElementItem, the value is expressed by one XML
	// element within the ElementItem element." Dropping it lost a rule's whole geometry.
	if got := list.ElementItem[0].Value; !strings.Contains(got, "PolygonConfiguration") {
		t.Errorf("ElementItem.Value = %q, want the element it contains", got)
	}
}

// tt:Message's attributes are unqualified, the property TestSimpleItemMarshalsUnqualified-
// Attributes already pins for SimpleItem and for the same reason: onvif.xsd declares them
// locally and sets no attributeFormDefault, so they belong to no namespace. A qualified
// UtcTime would not bind on the way in and would not be recognised on the way out.
func TestMessageAttributesAreUnqualified(t *testing.T) {
	const doc = `<Message xmlns="http://www.onvif.org/ver10/schema"
  UtcTime="2026-09-10T14:02:40Z" PropertyOperation="Initialized">
  <Data><SimpleItem Name="State" Value="true"/></Data>
</Message>`

	var message Message
	if err := xml.Unmarshal([]byte(doc), &message); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := string(message.UtcTime); got != "2026-09-10T14:02:40Z" {
		t.Errorf("Message.UtcTime = %q, want the stamp — a notification without it cannot be ordered", got)
	}
	if got := message.PropertyOperation; got != "Initialized" {
		t.Errorf("Message.PropertyOperation = %q, want Initialized (section 9.4.2)", got)
	}
	if got := len(message.Data.SimpleItem); got != 1 {
		t.Fatalf("Message.Data kept %d items, want 1", got)
	}

	b, err := xml.Marshal(message)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(b); strings.Contains(got, ":UtcTime=") || strings.Contains(got, ":PropertyOperation=") {
		t.Errorf("Message attributes are namespace-qualified, but the schema declares them local: %s", got)
	}
}

// An ordinary event is not a property event, so PropertyOperation must be absent rather than
// present and empty: section 9.4.2 gives it three values and no fourth meaning "not one".
func TestPropertyOperationIsOmittedOnAnOrdinaryEvent(t *testing.T) {
	b, err := xml.Marshal(Message{UtcTime: "2026-09-10T14:02:40Z"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(b); strings.Contains(got, "PropertyOperation") {
		t.Errorf("an ordinary event claims to be a property one: %s", got)
	}
}

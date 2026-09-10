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

// ONVIF requires WS-Addressing for the event service, and CallMethod emitted no wsa:Action
// for any operation. The strings under test are the soapAction literals of
// docs/wsdl/event.wsdl -- checked against the file itself, not against the operation name,
// because ONVIF's own port types and the OASIS bw-2 ones are spelled differently and
// deriving either from the Go type name would reconstruct the wrong one.

import (
	"bufio"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wsaActor is networking.WSAActor, restated here: importing networking from event would
// invert the dependency, since networking is the layer below.
type wsaActor interface {
	WSAAction() string
}

// TestEventActionsMatchTheWSDLBindings checks each action against the literal in the WSDL,
// naming the line it is on.
func TestEventActionsMatchTheWSDLBindings(t *testing.T) {
	for _, tc := range []struct {
		wsdlLine int
		request  wsaActor
		want     string
	}{
		{444, GetServiceCapabilities{}, "http://www.onvif.org/ver10/events/wsdl/EventPortType/GetServiceCapabilitiesRequest"},
		{453, CreatePullPointSubscription{}, "http://www.onvif.org/ver10/events/wsdl/EventPortType/CreatePullPointSubscriptionRequest"},
		{498, GetEventProperties{}, "http://www.onvif.org/ver10/events/wsdl/EventPortType/GetEventPropertiesRequest"},
		{411, PullMessages{}, "http://www.onvif.org/ver10/events/wsdl/PullPointSubscription/PullMessagesRequest"},
		{423, Seek{}, "http://www.onvif.org/ver10/events/wsdl/PullPointSubscription/SeekRequest"},
		{432, SetSynchronizationPoint{}, "http://www.onvif.org/ver10/events/wsdl/PullPointSubscription/SetSynchronizationPointRequest"},
		{543, Subscribe{}, "http://docs.oasis-open.org/wsn/bw-2/NotificationProducer/SubscribeRequest"},
		{510, Renew{}, "http://docs.oasis-open.org/wsn/bw-2/SubscriptionManager/RenewRequest"},
		{525, Unsubscribe{}, "http://docs.oasis-open.org/wsn/bw-2/SubscriptionManager/UnsubscribeRequest"},
	} {
		if got := tc.request.WSAAction(); got != tc.want {
			t.Errorf("%T.WSAAction() = %q, want %q (docs/wsdl/event.wsdl:%d)",
				tc.request, got, tc.want, tc.wsdlLine)
		}
	}
}

// TestEveryEventOperationDeclaresAnAction ties the actions to the calls.txt 1:1 invariant,
// so a tenth event operation cannot be added without one. The type is found by name rather
// than listed, which is what makes that true.
func TestEveryEventOperationDeclaresAnAction(t *testing.T) {
	byName := map[string]wsaActor{
		"GetServiceCapabilities":      GetServiceCapabilities{},
		"CreatePullPointSubscription": CreatePullPointSubscription{},
		"GetEventProperties":          GetEventProperties{},
		"PullMessages":                PullMessages{},
		"Seek":                        Seek{},
		"SetSynchronizationPoint":     SetSynchronizationPoint{},
		"Subscribe":                   Subscribe{},
		"Renew":                       Renew{},
		"Unsubscribe":                 Unsubscribe{},
	}

	f, err := os.Open(filepath.Join("calls.txt"))
	if err != nil {
		t.Fatalf("open calls.txt: %v", err)
	}
	defer f.Close()

	operations := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Same filter as bin/onvif-codegen.
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name := strings.Fields(line)[0]
		operations++

		request, known := byName[name]
		if !known {
			t.Errorf("calls.txt lists %s, which declares no wsa:Action; ONVIF requires "+
				"WS-Addressing for the event service", name)
			continue
		}
		if request.WSAAction() == "" {
			t.Errorf("%s.WSAAction() is empty", name)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read calls.txt: %v", err)
	}
	if operations != len(byName) {
		t.Errorf("calls.txt lists %d operations, %d are covered here", operations, len(byName))
	}
}

// TestRenewAndUnsubscribeMarshalInTheNotificationNamespace pins the missing XMLName tags.
// Without them xml.Marshal names the element after the Go type, so the requests went out as
// unqualified <Renew> and <Unsubscribe>. b-2.xsd declares both in
// http://docs.oasis-open.org/wsn/b-2 with elementFormDefault="qualified", and Subscribe
// beside them already had the tag -- which is why only these two were wrong.
func TestRenewAndUnsubscribeMarshalInTheNotificationNamespace(t *testing.T) {
	for _, tc := range []struct {
		request any
		want    string
	}{
		{Renew{}, "<wsnt:Renew>"},
		{Unsubscribe{}, "<wsnt:Unsubscribe>"},
	} {
		b, err := xml.Marshal(tc.request)
		if err != nil {
			t.Fatalf("Marshal %T: %v", tc.request, err)
		}
		got := string(b)

		if !strings.HasPrefix(got, tc.want) {
			t.Errorf("Marshal(%T) = %q, want it to start with %q", tc.request, got, tc.want)
		}
		// The unqualified name is what the device would have seen, so refuse it explicitly.
		bare := strings.Replace(tc.want, "wsnt:", "", 1)
		if strings.HasPrefix(got, bare) {
			t.Errorf("Marshal(%T) = %q emits the unqualified element", tc.request, got)
		}
	}

	// Unsubscribe carried an untagged Any string, which marshalled as a stray empty <Any>
	// inside the request body.
	b, err := xml.Marshal(Unsubscribe{})
	if err != nil {
		t.Fatalf("Marshal Unsubscribe: %v", err)
	}
	if strings.Contains(string(b), "<Any>") {
		t.Errorf("Marshal(Unsubscribe) = %q still emits the stray Any element", b)
	}
}

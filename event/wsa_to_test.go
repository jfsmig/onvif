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

// ONVIF addresses five of the nine event operations to the subscription manager that
// CreatePullPointSubscriptionResponse hands back, and four to the service. Nothing expressed
// that, so PullMessages and Unsubscribe went to the event service endpoint and a real
// subscription could not be completed -- which is what the comment in actions.go used to
// record as unfixed.
//
// The split is by port type and by nothing about the operation's name, so it is checked
// against the bindings in docs/wsdl/event.wsdl rather than derived: PullPointSubscription at
// :411, :423, :432 and the OASIS bw-2 SubscriptionManager at :510, :525 are addressed to the
// manager, while EventPortType at :444, :453, :498 and the bw-2 NotificationProducer at :543
// are addressed to the service. ONVIF Core section 9.10.5 shows the first kind carrying
// wsa:To; section 9.10.3 shows the second kind carrying none.

import (
	"bufio"
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

// wsaAddressee is networking.WSAAddressee, restated here for the reason wsaActor is:
// importing networking from event would invert the dependency, since networking is the layer
// below.
type wsaAddressee interface {
	WSATo() string
}

// TestOnlyManagerAddressedOperationsDeclareADestination ties the split to the calls.txt 1:1
// invariant, so a tenth event operation cannot be added without a decision about which side
// of it the operation falls on. The types are found by name rather than listed, which is
// what makes that true.
func TestOnlyManagerAddressedOperationsDeclareADestination(t *testing.T) {
	// The request value for each name, and the WSDL line of its binding.
	byName := map[string]struct {
		request  any
		wsdlLine int
		manager  bool
	}{
		"PullMessages":                {PullMessages{}, 411, true},
		"Seek":                        {Seek{}, 423, true},
		"SetSynchronizationPoint":     {SetSynchronizationPoint{}, 432, true},
		"Renew":                       {Renew{}, 510, true},
		"Unsubscribe":                 {Unsubscribe{}, 525, true},
		"GetServiceCapabilities":      {GetServiceCapabilities{}, 444, false},
		"CreatePullPointSubscription": {CreatePullPointSubscription{}, 453, false},
		"GetEventProperties":          {GetEventProperties{}, 498, false},
		"Subscribe":                   {Subscribe{}, 543, false},
	}

	f, err := os.Open("calls.txt")
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

		operation, known := byName[name]
		if !known {
			t.Errorf("calls.txt lists %s, and nothing here says whether ONVIF addresses it "+
				"to a subscription manager or to the event service", name)
			continue
		}

		_, declares := operation.request.(wsaAddressee)
		switch {
		case operation.manager && !declares:
			t.Errorf("%s is on a manager-addressed port type (docs/wsdl/event.wsdl:%d) but "+
				"declares no destination, so it would be POSTed to the event service and "+
				"reach no subscription", name, operation.wsdlLine)
		case !operation.manager && declares:
			t.Errorf("%s is addressed to the event service (docs/wsdl/event.wsdl:%d) but "+
				"declares a destination; the request in ONVIF Core section 9.10.3 carries "+
				"no wsa:To", name, operation.wsdlLine)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read calls.txt: %v", err)
	}
	if operations != len(byName) {
		t.Errorf("calls.txt lists %d operations, %d are covered here", operations, len(byName))
	}
}

// The receiver has to be a value, and that is load-bearing rather than stylistic: CallMethod
// is handed an interface holding a value, because the generated wrapper passes
// `request PullMessages` by value, so a pointer receiver would not satisfy the assertion and
// the request would quietly go to the event service endpoint instead.
func TestADestinationIsDeclaredOnTheValue(t *testing.T) {
	var (
		_ wsaAddressee = PullMessages{}
		_ wsaAddressee = Seek{}
		_ wsaAddressee = SetSynchronizationPoint{}
		_ wsaAddressee = Renew{}
		_ wsaAddressee = Unsubscribe{}
	)
}

// The destination is where the message goes, not one of its parts, so it must not appear in
// the body. That is what the xml:"-" tag on the To fields is for, and it is one character
// away from being a field a device would reject.
func TestTheDestinationNeverReachesTheWire(t *testing.T) {
	const manager = "http://192.168.1.70:8000/onvif/Subscription?Idx=7"

	for _, request := range []any{
		PullMessages{To: manager},
		Seek{To: manager},
		SetSynchronizationPoint{To: manager},
		Renew{To: manager},
		Unsubscribe{To: manager},
	} {
		b, err := xml.Marshal(request)
		if err != nil {
			t.Fatalf("Marshal %T: %v", request, err)
		}
		got := string(b)
		if strings.Contains(got, "Subscription?Idx=7") {
			t.Errorf("Marshal(%T) puts the subscription URI in the body: %s", request, got)
		}
		if strings.Contains(got, "<To>") {
			t.Errorf("Marshal(%T) emits a stray To element: %s", request, got)
		}
	}
}

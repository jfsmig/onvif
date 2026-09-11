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

// wsa:Action for the event operations.
//
// ONVIF requires WS-Addressing for the event service, and CallMethod emitted no
// wsa:Action for any operation, so every event request went out without the header that
// identifies it. Only this package declares actions: the device, media and PTZ examples in
// the specification carry no addressing header, and adding one to all 205 operations would
// be unrequested risk against devices that work today. networking.WSAActor is the opt-in.
//
// The strings are the soapAction literals of docs/wsdl/event.wsdl, copied rather than
// derived from the operation name, because the two groups are spelled differently: ONVIF's
// own port types live under http://www.onvif.org/ver10/events/wsdl/, while Subscribe, Renew
// and Unsubscribe come from the OASIS bw-2 port types. Each line names the WSDL line it
// came from.
//
// The action alone is not enough for five of the nine, and the WSATo methods at the bottom
// of this file are the other half. ONVIF addresses the operations of the
// PullPointSubscription and bw-2 SubscriptionManager port types to the subscription manager
// returned in CreatePullPointSubscriptionResponse: they carry a wsa:To of that URI and are
// POSTed there rather than to the event service endpoint. Compare the requests in ONVIF Core
// sections 9.10.5 and 9.10.7 with the one in 9.10.3, which has no wsa:To at all.
//
// So the split below is by port type and not by anything about the operation's name, and the
// four operations of EventPortType and of the bw-2 NotificationProducer must NOT declare a
// destination. event/wsa_to_test.go reads that grouping out of calls.txt and holds it, which
// is what puts a tenth operation on the right side of it.
const (
	// docs/wsdl/event.wsdl:444
	actionGetServiceCapabilities = "http://www.onvif.org/ver10/events/wsdl/EventPortType/GetServiceCapabilitiesRequest"
	// docs/wsdl/event.wsdl:453
	actionCreatePullPointSubscription = "http://www.onvif.org/ver10/events/wsdl/EventPortType/CreatePullPointSubscriptionRequest"
	// docs/wsdl/event.wsdl:498
	actionGetEventProperties = "http://www.onvif.org/ver10/events/wsdl/EventPortType/GetEventPropertiesRequest"
	// docs/wsdl/event.wsdl:411
	actionPullMessages = "http://www.onvif.org/ver10/events/wsdl/PullPointSubscription/PullMessagesRequest"
	// docs/wsdl/event.wsdl:423
	actionSeek = "http://www.onvif.org/ver10/events/wsdl/PullPointSubscription/SeekRequest"
	// docs/wsdl/event.wsdl:432
	actionSetSynchronizationPoint = "http://www.onvif.org/ver10/events/wsdl/PullPointSubscription/SetSynchronizationPointRequest"
	// docs/wsdl/event.wsdl:543 -- OASIS bw-2, not ONVIF
	actionSubscribe = "http://docs.oasis-open.org/wsn/bw-2/NotificationProducer/SubscribeRequest"
	// docs/wsdl/event.wsdl:510 -- OASIS bw-2, not ONVIF
	actionRenew = "http://docs.oasis-open.org/wsn/bw-2/SubscriptionManager/RenewRequest"
	// docs/wsdl/event.wsdl:525 -- OASIS bw-2, not ONVIF
	actionUnsubscribe = "http://docs.oasis-open.org/wsn/bw-2/SubscriptionManager/UnsubscribeRequest"
)

func (GetServiceCapabilities) WSAAction() string      { return actionGetServiceCapabilities }
func (CreatePullPointSubscription) WSAAction() string { return actionCreatePullPointSubscription }
func (GetEventProperties) WSAAction() string          { return actionGetEventProperties }
func (PullMessages) WSAAction() string                { return actionPullMessages }
func (Seek) WSAAction() string                        { return actionSeek }
func (SetSynchronizationPoint) WSAAction() string     { return actionSetSynchronizationPoint }
func (Subscribe) WSAAction() string                   { return actionSubscribe }
func (Renew) WSAAction() string                       { return actionRenew }
func (Unsubscribe) WSAAction() string                 { return actionUnsubscribe }

// The subscription manager each of these is addressed to, implementing
// networking.WSAAddressee. The URI is carried in a To field tagged xml:"-", so it never
// reaches the body: it is where the message goes, not one of its parts.
//
// Value receivers, and that is load-bearing rather than stylistic. CallMethod is handed an
// interface holding a value -- the generated wrapper passes `request PullMessages` by value
// -- so a pointer receiver would not satisfy the assertion and the request would quietly go
// to the event service endpoint instead, which is the failure this whole mechanism exists to
// remove.
//
// The remaining four operations deliberately have no such method: GetServiceCapabilities,
// CreatePullPointSubscription and GetEventProperties belong to EventPortType and Subscribe to
// the bw-2 NotificationProducer, and all four are addressed to the service.
func (r PullMessages) WSATo() string            { return r.To }
func (r Seek) WSATo() string                    { return r.To }
func (r SetSynchronizationPoint) WSATo() string { return r.To }
func (r Renew) WSATo() string                   { return r.To }
func (r Unsubscribe) WSATo() string             { return r.To }

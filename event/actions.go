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
// What this does NOT fix: ONVIF addresses PullMessages, Renew and Unsubscribe to the
// subscription manager returned in CreatePullPointSubscriptionResponse -- they need a
// wsa:To of that URI and must be POSTed there, not to the event service endpoint.
// CallMethod routes on the request struct's package name and has no way to express a
// per-request target, so those three still cannot complete a real subscription. The header
// is correct now; the subscription flow remains separate work.
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

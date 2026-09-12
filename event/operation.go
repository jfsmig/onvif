// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// Portions of this file derive from the goonvif project, now use-go/onvif,
// originally distributed under the MIT License, see LICENSE.MIT,
// Copyright (c) 2018 Yakovlev Dmitry, Zhorzh Palanjyan, Crazybber.
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
	"github.com/jfsmig/onvif/v2/xsd"
)

// GetServiceCapabilities action
type GetServiceCapabilities struct {
	XMLName string `xml:"tev:GetServiceCapabilities"`
}

// GetServiceCapabilitiesResponse type
type GetServiceCapabilitiesResponse struct {
	Capabilities Capabilities
}

// SubscriptionPolicy action
type SubscriptionPolicy struct { //tev http://www.onvif.org/ver10/events/wsdl
	ChangedOnly xsd.Boolean `xml:"ChangedOnly,attr"`
}

// Subscribe action for subscribe event topic
type Subscribe struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	XMLName                struct{}              `xml:"wsnt:Subscribe"`
	ConsumerReference      EndpointReferenceType `xml:"wsnt:ConsumerReference"`
	Filter                 FilterType            `xml:"wsnt:Filter"`
	SubscriptionPolicy     SubscriptionPolicy    `xml:"wsnt:SubscriptionPolicy"`
	InitialTerminationTime TerminationTime       `xml:"wsnt:InitialTerminationTime"`
}

// SubscribeResponse message for subscribe event topic
type SubscribeResponse struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	// SubscriptionReference, not ConsumerReference: b-2.xsd declares the response's local
	// element as SubscriptionReference (ConsumerReference belongs to the Subscribe request).
	// The old field carried the wrong name and an unmatchable tag, so it never bound.
	SubscriptionReference EndpointReferenceType `xml:"http://docs.oasis-open.org/wsn/b-2 SubscriptionReference"`
	CurrentTime           CurrentTime           `xml:"http://docs.oasis-open.org/wsn/b-2 CurrentTime"`
	TerminationTime       TerminationTime       `xml:"http://docs.oasis-open.org/wsn/b-2 TerminationTime"`
}

// Renew action for refresh event topic subscription
//
// XMLName, like Subscribe above: without it xml.Marshal names the element after the Go
// type, so the request went out as an unqualified <Renew> instead of <wsnt:Renew>. b-2.xsd
// declares Renew in http://docs.oasis-open.org/wsn/b-2 with elementFormDefault="qualified",
// so an unqualified element is a different element as far as the device is concerned.
type Renew struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	XMLName         struct{}                   `xml:"wsnt:Renew"`
	TerminationTime AbsoluteOrRelativeTimeType `xml:"wsnt:TerminationTime"`

	// To addresses the request to a subscription manager; see actions.go.
	To string `xml:"-"`
}

// RenewResponse for Renew action
type RenewResponse struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	TerminationTime TerminationTime `xml:"http://docs.oasis-open.org/wsn/b-2 TerminationTime"`
	CurrentTime     CurrentTime     `xml:"http://docs.oasis-open.org/wsn/b-2 CurrentTime"`
}

// Unsubscribe action for Unsubscribe event topic
//
// Same missing XMLName as Renew had. The Any field went with it: b-2.xsd gives Unsubscribe
// an optional xs:any, and an untagged string field marshalled as a stray empty <Any>
// element inside the request body.
type Unsubscribe struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	XMLName struct{} `xml:"wsnt:Unsubscribe"`

	// To addresses the request to a subscription manager; see actions.go.
	To string `xml:"-"`
}

// UnsubscribeResponse message for Unsubscribe event topic
type UnsubscribeResponse struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	Any string
}

// CreatePullPointSubscription action.
//
// Every child is minOccurs="0" (docs/wsdl/event.wsdl:79-98) and every one is a pointer, so
// that an unset one is left out of the request. encoding/xml has no other way to say that:
// its omitempty ignores struct kinds -- isEmptyValue handles strings, slices, maps, the
// numerics, interfaces and pointers, and nothing else -- so a value field emitted its
// element whatever it held. The zero request therefore went out carrying an empty
// <wsnt:TopicExpression Dialect="">, which is a topic expression in no dialect and which b-2
// answers with a fault, and that is the one request that starts a subscription.
//
// ONVIF Core section 9.1.1 is what makes the empty request meaningful: "If no Filter element
// is specified the pullpoint shall notify all occurring events to the client." The reference
// request in section 9.10.3 carries a Filter and an InitialTerminationTime and no
// SubscriptionPolicy at all.
//
// The two tag forms here are not interchangeable and both are deliberate. The space form
// emits xmlns="...events/wsdl" as a *default* namespace on the element, which an unprefixed
// child would inherit; it is safe only because these two carry character data and an
// attribute and never a child element.
type CreatePullPointSubscription struct {
	XMLName                string                      `xml:"tev:CreatePullPointSubscription"`
	Filter                 *FilterType                 `xml:"tev:Filter"`
	InitialTerminationTime *AbsoluteOrRelativeTimeType `xml:"http://www.onvif.org/ver10/events/wsdl InitialTerminationTime"`
	SubscriptionPolicy     *SubscriptionPolicy         `xml:"http://www.onvif.org/ver10/events/wsdl SubscriptionPolicy"`
}

// CreatePullPointSubscriptionResponse action.
//
// Untagged, like PullMessagesResponse and for the same reason -- and here it is load-bearing
// rather than merely tolerant. docs/wsdl/event.wsdl:106 declares SubscriptionReference as a
// local element of the tev schema, so section 9.10.4 shows <tet:SubscriptionReference>, while
// b-2 declares SubscribeResponse's in wsnt. One untagged field binds both. This is the field
// the whole event flow hangs on, and event/namespace_test.go pins it.
type CreatePullPointSubscriptionResponse struct {
	SubscriptionReference EndpointReferenceType
	CurrentTime           CurrentTime
	TerminationTime       TerminationTime
}

// GetEventProperties action
type GetEventProperties struct {
	XMLName string `xml:"tev:GetEventProperties"`
}

// GetEventPropertiesResponse action
type GetEventPropertiesResponse struct {
	TopicNamespaceLocation          xsd.AnyURI
	FixedTopicSet                   FixedTopicSet
	TopicSet                        TopicSet
	TopicExpressionDialect          TopicExpressionDialect
	MessageContentFilterDialect     xsd.AnyURI
	ProducerPropertiesFilterDialect xsd.AnyURI
	MessageContentSchemaLocation    xsd.AnyURI
}

//Port type PullPointSubscription

// PullMessages Action
type PullMessages struct {
	XMLName      string       `xml:"tev:PullMessages"`
	Timeout      xsd.Duration `xml:"tev:Timeout"`
	MessageLimit xsd.Int      `xml:"tev:MessageLimit"`

	// To addresses the request to a subscription manager; see actions.go.
	To string `xml:"-"`
}

// PullMessagesResponse response type.
//
// NotificationMessage is a slice: docs/wsdl/event.wsdl:156 declares it minOccurs="0"
// maxOccurs="unbounded", ONVIF Core section 9.1.2 Table 81 says [0][unbounded], and the
// reference response in section 9.10.6 carries two. It was a single value, so encoding/xml
// assigned each match to the same field in turn and only the last survived -- and with a
// MessageLimit above one, several messages in a reply is the normal case rather than the
// exception.
//
// The fields stay untagged, and that absence is the decision. Binding on the local name
// accepts whatever namespace the device qualified the element with, which here is both the
// tolerant and the correct answer: the same two local names live in two namespaces across
// this one flow -- CurrentTime and TerminationTime are local tev elements in this response
// (docs/wsdl/event.wsdl:146,151) but wsnt refs in CreatePullPointSubscriptionResponse
// (:111,116), exactly as the examples in sections 9.10.6 and 9.10.4 show. The local names
// within each parent are unique, so a namespace in the tag would add no discrimination and
// could only turn a device that qualifies differently from the WSDL into a silently empty
// result. Requests are the opposite case: there the tag is what puts the element into a
// namespace, so it has to be exact.
type PullMessagesResponse struct {
	CurrentTime         CurrentTime
	TerminationTime     TerminationTime
	NotificationMessage []NotificationMessage
}

// PullMessagesFaultResponse response type
type PullMessagesFaultResponse struct {
	MaxTimeout      xsd.Duration
	MaxMessageLimit xsd.Int
}

// Seek action
type Seek struct {
	XMLName string       `xml:"tev:Seek"`
	UtcTime xsd.DateTime `xml:"tev:UtcTime"`
	Reverse xsd.Boolean  `xml:"tev:Reverse"`

	// To addresses the request to a subscription manager; see actions.go.
	To string `xml:"-"`
}

// SeekResponse action
type SeekResponse struct {
}

// SetSynchronizationPoint action
type SetSynchronizationPoint struct {
	XMLName string `xml:"tev:SetSynchronizationPoint"`

	// To addresses the request to a subscription manager; see actions.go.
	To string `xml:"-"`
}

// SetSynchronizationPointResponse action
type SetSynchronizationPointResponse struct {
}

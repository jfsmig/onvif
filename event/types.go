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
	"github.com/jfsmig/onvif/xsd"
	"github.com/jfsmig/onvif/xsd/onvif"
)

//go:generate go run github.com/jfsmig/onvif/bin/onvif-codegen sdk event calls.txt

// Address Alias
type Address xsd.String

// CurrentTime alias
type CurrentTime xsd.DateTime //wsnt http://docs.oasis-open.org/wsn/b-2.xsd
// TerminationTime alias
type TerminationTime xsd.DateTime //wsnt http://docs.oasis-open.org/wsn/b-2.xsd
// FixedTopicSet alias
type FixedTopicSet xsd.Boolean //wsnt http://docs.oasis-open.org/wsn/b-2.xsd

// Documentation alias
type Documentation xsd.AnyType //wstop http://docs.oasis-open.org/wsn/t-1.xsd

// TopicExpressionDialect alias
type TopicExpressionDialect xsd.AnyURI

// ActionType for AttributedURIType
type ActionType AttributedURIType

// AttributedURIType in ws-addr
type AttributedURIType xsd.AnyURI //wsa https://www.w3.org/2005/08/addressing/ws-addr.xsd

// AbsoluteOrRelativeTimeType is b-2's <xsd:union memberTypes="xsd:dateTime xsd:duration"/>.
//
// A union has no structure: its lexical space is that of either member, so the Go type is
// the string carrying whichever form the caller chose -- "PT1M" or "2026-08-31T10:05:00Z".
// It used to embed xsd.DateTime and xsd.Duration, and encoding/xml does not flatten an
// anonymous field whose type is not a struct -- getTypeInfo only recurses into a struct, and
// otherwise names the element after the field, which for an embedded field is its type. So
// both InitialTerminationTime and wsnt:TerminationTime went out carrying <DateTime/> and
// <Duration/> children, which appear in no schema and which a union admits none of. This is
// the BUG(r) marker that stood on CreatePullPointSubscription.
type AbsoluteOrRelativeTimeType xsd.AnySimpleType //wsnt http://docs.oasis-open.org/wsn/b-2.xsd

// EndpointReferenceType in ws-addr.
//
// The children live in the WS-Addressing namespace, not WS-Notification: the tags used to
// read wsnt:, which was wrong twice over — Go's tag separator is a space, not a colon, so
// the whole string became the element name and a device's <wsa:Address> never bound. That
// left CreatePullPointSubscriptionResponse.SubscriptionReference.Address empty, and with it
// the pull-point flow.
type EndpointReferenceType struct { //wsa http://www.w3.org/2005/08/addressing/ws-addr.xsd
	Address             AttributedURIType       `xml:"http://www.w3.org/2005/08/addressing Address"`
	ReferenceParameters ReferenceParametersType `xml:"http://www.w3.org/2005/08/addressing ReferenceParameters"`
	Metadata            MetadataType            `xml:"http://www.w3.org/2005/08/addressing Metadata"`
}

// FilterType is b-2's wsnt:FilterType: a filter carries the expressions the subscriber
// wants and nothing else.
//
// Both fields are pointers, for the reason CreatePullPointSubscription's children are: a
// caller filtering on a topic alone must not also send an empty message-content filter,
// which is not a filter any dialect accepts. ONVIF Core section 9.10.3 shows both present,
// section 9.6.3 lists the dialects, and Dialect is what an empty expression lacks.
//
// The b-2 Subscribe request in operation.go still holds this by value, so it still marshals
// an empty wsnt:Filter, and that is deliberate rather than an oversight: this library
// implements the real-time pull-point interface only -- push notification would need an
// inbound listener, which does not exist here -- so Subscribe has no caller to be wrong for,
// and TestSubscribeRequestWireFormatUnchanged pins its bytes on purpose. Give it a pointer
// when something first sends it, against a camera.
type FilterType struct {
	TopicExpression *TopicExpressionType `xml:"http://docs.oasis-open.org/wsn/b-2 TopicExpression"`
	MessageContent  *QueryExpressionType `xml:"http://docs.oasis-open.org/wsn/b-2 MessageContent"`
}

// EndpointReference alais
type EndpointReference EndpointReferenceType

// ReferenceParametersType in ws-addr
type ReferenceParametersType struct { //wsa https://www.w3.org/2005/08/addressing/ws-addr.xsd
	//Here can be anyAttribute
}

// Metadata in ws-addr
type Metadata MetadataType //wsa https://www.w3.org/2005/08/addressing/ws-addr.xsd

// MetadataType in ws-addr
type MetadataType struct { //wsa https://www.w3.org/2005/08/addressing/ws-addr.xsd

	//Here can be anyAttribute
}

// TopicSet alias
type TopicSet TopicSetType //wstop http://docs.oasis-open.org/wsn/t-1.xsd

// TopicSetType alias
type TopicSetType struct { //wstop http://docs.oasis-open.org/wsn/t-1.xsd
	ExtensibleDocumented
	//here can be any element
}

// ExtensibleDocumented struct
type ExtensibleDocumented struct { //wstop http://docs.oasis-open.org/wsn/t-1.xsd
	// Tagged, because the element is spelled "documentation" in lower case --
	// xsd/onvif/t-1.xsd:10 -- and encoding/xml matches a field name against an element name
	// case-sensitively. Untagged, this field therefore bound to nothing: a device's topic
	// documentation was parsed, matched no field, and was dropped without a word. The
	// upstream comment here flagged the lower-case spelling and stopped short of the
	// consequence.
	Documentation Documentation `xml:"documentation"`
	//here can be anyAttribute
}

// ProducerReference Alias
type ProducerReference EndpointReferenceType

// SubscriptionReference Alias
type SubscriptionReference EndpointReferenceType

// NotificationMessageHolderType is b-2's holder for one notification, the element a
// PullMessagesResponse repeats (ONVIF Core section 9.4).
//
// The fields are untagged, which binds them on the local name whatever namespace the device
// qualified them with. That is deliberate here and the reasoning is PullMessagesResponse's;
// the local names within this holder are unique, so a namespace would add no discrimination
// and could only turn a device that qualifies differently into a silently empty result.
type NotificationMessageHolderType struct {
	SubscriptionReference SubscriptionReference //wsnt http://docs.oasis-open.org/wsn/b-2.xsd
	Topic                 Topic
	ProducerReference     ProducerReference
	Message               MessageHolder
}

// MessageHolder is the anonymous complexType of wsnt:NotificationMessage/wsnt:Message, whose
// children are the ONVIF payload.
//
// It replaces a field whose type was Message = xsd.AnyType, that is a string, so
// encoding/xml bound the character data of <wsnt:Message> -- whitespace, since its content is
// an element and not text. Every notification therefore arrived with its topic and without
// its event, which is the whole of what a subscriber wanted.
//
// Message is a slice because ONVIF Core section 9.4 (page 107) restates b-2's
// NotificationMessageHolderType with
//
//	<tt:element name="Message" type="tt:Message" maxOccurs="unbounded"/>
//
// and says in prose that the holder carries "one or more notification messages of type
// tt:Message". b-2.xsd is not vendored in this tree, so section 9.4 is the citation.
//
// A single value here did not merely drop the extra messages, it fabricated one: encoding/xml
// assigned each tt:Message to the same field in turn, so UtcTime and PropertyOperation came
// from the LAST of them while Source, Key and Data -- slices, since ItemList went plural --
// appended across all of them. One record went out stamped with the second event's time
// carrying the first event's data items, which is an event that never occurred rather than an
// event that went missing.
type MessageHolder struct {
	Message []onvif.Message
}

// NotificationMessage Alias
type NotificationMessage NotificationMessageHolderType //wsnt http://docs.oasis-open.org/wsn/b-2.xsd

// QueryExpressionType struct for wsnt:MessageContent
type QueryExpressionType struct { //wsnt http://docs.oasis-open.org/wsn/b-2.xsd
	Dialect     xsd.AnyURI `xml:"Dialect,attr"`
	MessageKind xsd.String `xml:",chardata"` // boolean(ncex:Producer="15")
}

// MessageContentType Alias
type MessageContentType QueryExpressionType

// QueryExpression Alias
type QueryExpression QueryExpressionType

// TopicExpressionType struct for wsnt:TopicExpression
type TopicExpressionType struct { //wsnt http://docs.oasis-open.org/wsn/b-2.xsd
	Dialect    xsd.AnyURI `xml:"Dialect,attr"`
	TopicKinds xsd.String `xml:",chardata"`
}

// Topic Alias
type Topic TopicExpressionType

// Capabilities of event
type Capabilities struct { //tev
	WSSubscriptionPolicySupport                   xsd.Boolean `xml:"WSSubscriptionPolicySupport,attr"`
	WSPullPointSupport                            xsd.Boolean `xml:"WSPullPointSupport,attr"`
	WSPausableSubscriptionManagerInterfaceSupport xsd.Boolean `xml:"WSPausableSubscriptionManagerInterfaceSupport,attr"`
	MaxNotificationProducers                      xsd.Int     `xml:"MaxNotificationProducers,attr"`
	MaxPullPoints                                 xsd.Int     `xml:"MaxPullPoints,attr"`
	PersistentNotificationStorage                 xsd.Boolean `xml:"PersistentNotificationStorage,attr"`
}

// ResourceUnknownFault response type
type ResourceUnknownFault struct {
}

// InvalidFilterFault response type
type InvalidFilterFault struct {
}

// TopicExpressionDialectUnknownFault response type
type TopicExpressionDialectUnknownFault struct {
}

// InvalidTopicExpressionFault response type
type InvalidTopicExpressionFault struct {
}

// TopicNotSupportedFault response type
type TopicNotSupportedFault struct {
}

// InvalidProducerPropertiesExpressionFault response type
type InvalidProducerPropertiesExpressionFault struct {
}

// InvalidMessageContentExpressionFault response type
type InvalidMessageContentExpressionFault struct {
}

// UnacceptableInitialTerminationTimeFault response type
type UnacceptableInitialTerminationTimeFault struct {
}

// UnrecognizedPolicyRequestFault response type
type UnrecognizedPolicyRequestFault struct {
}

// UnsupportedPolicyRequestFault response type
type UnsupportedPolicyRequestFault struct {
}

// NotifyMessageNotSupportedFault response type
type NotifyMessageNotSupportedFault struct {
}

// SubscribeCreationFailedFault response type
type SubscribeCreationFailedFault struct {
}

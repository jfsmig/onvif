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
}

// UnsubscribeResponse message for Unsubscribe event topic
type UnsubscribeResponse struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	Any string
}

// CreatePullPointSubscription action
// BUG(r) Bad AbsoluteOrRelativeTimeType type
type CreatePullPointSubscription struct {
	XMLName                string                     `xml:"tev:CreatePullPointSubscription"`
	Filter                 FilterType                 `xml:"tev:Filter"`
	InitialTerminationTime AbsoluteOrRelativeTimeType `xml:"http://www.onvif.org/ver10/events/wsdl InitialTerminationTime"`
	SubscriptionPolicy     SubscriptionPolicy         `xml:"http://www.onvif.org/ver10/events/wsdl SubscriptionPolicy"`
}

// CreatePullPointSubscriptionResponse action
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
}

// PullMessagesResponse response type
type PullMessagesResponse struct {
	CurrentTime         CurrentTime
	TerminationTime     TerminationTime
	NotificationMessage NotificationMessage
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
}

// SeekResponse action
type SeekResponse struct {
}

// SetSynchronizationPoint action
type SetSynchronizationPoint struct {
	XMLName string `xml:"tev:SetSynchronizationPoint"`
}

// SetSynchronizationPointResponse action
type SetSynchronizationPointResponse struct {
}

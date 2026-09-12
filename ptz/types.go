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

package ptz

import (
	"github.com/jfsmig/onvif/xsd"
	"github.com/jfsmig/onvif/xsd/onvif"
)

//go:generate go run github.com/jfsmig/onvif/bin/onvif-codegen sdk ptz calls.txt

type Capabilities struct {
	EFlip                       xsd.Boolean `xml:"EFlip,attr"`
	Reverse                     xsd.Boolean `xml:"Reverse,attr"`
	GetCompatibleConfigurations xsd.Boolean `xml:"GetCompatibleConfigurations,attr"`
	MoveStatus                  xsd.Boolean `xml:"MoveStatus,attr"`
	StatusPosition              xsd.Boolean `xml:"StatusPosition,attr"`
}

//PTZ main types

type GetServiceCapabilities struct {
	XMLName string `xml:"tptz:GetServiceCapabilities"`
}

type GetServiceCapabilitiesResponse struct {
	Capabilities Capabilities
}

type GetNodes struct {
	XMLName string `xml:"tptz:GetNodes"`
}

type GetNodesResponse struct {
	PTZNode []onvif.PTZNode
}

type GetNode struct {
	XMLName   string               `xml:"tptz:GetNode"`
	NodeToken onvif.ReferenceToken `xml:"tptz:NodeToken"`
}

type GetNodeResponse struct {
	PTZNode onvif.PTZNode
}

type GetConfiguration struct {
	XMLName string `xml:"tptz:GetConfiguration"`
	// ptz.wsdl names this child PTZConfigurationToken, and it is a PTZ *configuration*
	// token, not a profile token. It used to be both misnamed and mistagged, so the device
	// received no token it recognised.
	PTZConfigurationToken onvif.ReferenceToken `xml:"tptz:PTZConfigurationToken"`
}

type GetConfigurationResponse struct {
	PTZConfiguration onvif.PTZConfiguration
}

type GetConfigurations struct {
	XMLName string `xml:"tptz:GetConfigurations"`
}

type GetConfigurationsResponse struct {
	PTZConfiguration []onvif.PTZConfiguration
}

type SetConfiguration struct {
	XMLName          string                 `xml:"tptz:SetConfiguration"`
	PTZConfiguration onvif.PTZConfiguration `xml:"tptz:PTZConfiguration"`
	ForcePersistence xsd.Boolean            `xml:"tptz:ForcePersistence"`
}

type SetConfigurationResponse struct {
}

type GetConfigurationOptions struct {
	XMLName string `xml:"tptz:GetConfigurationOptions"`
	// ptz.wsdl names this child ConfigurationToken -- note it differs from
	// GetConfiguration's PTZConfigurationToken -- and it too takes a configuration token.
	ConfigurationToken onvif.ReferenceToken `xml:"tptz:ConfigurationToken"`
}

type GetConfigurationOptionsResponse struct {
	PTZConfigurationOptions onvif.PTZConfigurationOptions
}

type SendAuxiliaryCommand struct {
	XMLName       string               `xml:"tptz:SendAuxiliaryCommand"`
	ProfileToken  onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	AuxiliaryData onvif.AuxiliaryData  `xml:"tptz:AuxiliaryData"`
}

type SendAuxiliaryCommandResponse struct {
	AuxiliaryResponse onvif.AuxiliaryData
}

type GetPresets struct {
	XMLName      string               `xml:"tptz:GetPresets"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

type GetPresetsResponse struct {
	Preset []onvif.PTZPreset
}

type SetPreset struct {
	XMLName      string               `xml:"tptz:SetPreset"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	PresetName   xsd.String           `xml:"tptz:PresetName"`
	PresetToken  onvif.ReferenceToken `xml:"tptz:PresetToken,omitempty"`
}

type SetPresetResponse struct {
	PresetToken onvif.ReferenceToken
}

type RemovePreset struct {
	XMLName      string               `xml:"tptz:RemovePreset"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	PresetToken  onvif.ReferenceToken `xml:"tptz:PresetToken"`
}

type RemovePresetResponse struct {
}

type GotoPreset struct {
	XMLName      string               `xml:"tptz:GotoPreset"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	PresetToken  onvif.ReferenceToken `xml:"tptz:PresetToken"`
	Speed        *onvif.PTZSpeed      `xml:"tptz:Speed,omitempty"`
}

type GotoPresetResponse struct {
}

type GotoHomePosition struct {
	XMLName      string               `xml:"tptz:GotoHomePosition"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	Speed        *onvif.PTZSpeed      `xml:"tptz:Speed,omitempty"`
}

type GotoHomePositionResponse struct {
}

type SetHomePosition struct {
	XMLName      string               `xml:"tptz:SetHomePosition"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

type SetHomePositionResponse struct {
}

type ContinuousMove struct {
	XMLName      string               `xml:"tptz:ContinuousMove"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	Velocity     onvif.PTZSpeed       `xml:"tptz:Velocity"`
	Timeout      xsd.Duration         `xml:"tptz:Timeout,omitempty"`
}

type ContinuousMoveResponse struct {
}

type RelativeMove struct {
	XMLName      string               `xml:"tptz:RelativeMove"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	Translation  onvif.PTZVector      `xml:"tptz:Translation"`
	Speed        *onvif.PTZSpeed      `xml:"tptz:Speed,omitempty"`
}

type RelativeMoveResponse struct {
}

type GetStatus struct {
	XMLName      string               `xml:"tptz:GetStatus"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

type GetStatusResponse struct {
	PTZStatus onvif.PTZStatus
}

type AbsoluteMove struct {
	XMLName      string               `xml:"tptz:AbsoluteMove"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	Position     onvif.PTZVector      `xml:"tptz:Position"`
	Speed        *onvif.PTZSpeed      `xml:"tptz:Speed,omitempty"`
}

type AbsoluteMoveResponse struct {
}

type GeoMove struct {
	XMLName      string               `xml:"tptz:GeoMove"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	Target       onvif.GeoLocation    `xml:"tptz:Target"`
	Speed        *onvif.PTZSpeed      `xml:"tptz:Speed,omitempty"`
	AreaHeight   xsd.Float            `xml:"tptz:AreaHeight,omitempty"`
	AreaWidth    xsd.Float            `xml:"tptz:AreaWidth,omitempty"`
}

type GeoMoveResponse struct {
}

// Stop halts the axes it names, and every axis when it names none.
//
// PanTilt and Zoom are pointers because docs/wsdl/ptz.wsdl:540 gives both minOccurs="0" and
// makes the absence, not the value, the instruction: "If PanTilt arguments are not present,
// this command stops these movements." So nil stops the axis, false explicitly leaves it
// running, and a plain bool could not hold the difference -- the zero Stop used to send
// false for both, which is a well-formed request to stop nothing at all.
type Stop struct {
	XMLName      string               `xml:"tptz:Stop"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	PanTilt      *xsd.Boolean         `xml:"tptz:PanTilt,omitempty"`
	Zoom         *xsd.Boolean         `xml:"tptz:Zoom,omitempty"`
}

type StopResponse struct {
}

type GetPresetTours struct {
	XMLName      string               `xml:"tptz:GetPresetTours"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

type GetPresetToursResponse struct {
	PresetTour []onvif.PresetTour
}

type GetPresetTour struct {
	XMLName         string               `xml:"tptz:GetPresetTour"`
	ProfileToken    onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	PresetTourToken onvif.ReferenceToken `xml:"tptz:PresetTourToken"`
}

type GetPresetTourResponse struct {
	PresetTour onvif.PresetTour
}

type GetPresetTourOptions struct {
	XMLName         string               `xml:"tptz:GetPresetTourOptions"`
	ProfileToken    onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	PresetTourToken onvif.ReferenceToken `xml:"tptz:PresetTourToken"`
}

type GetPresetTourOptionsResponse struct {
	Options onvif.PTZPresetTourOptions
}

type CreatePresetTour struct {
	XMLName      string               `xml:"tptz:CreatePresetTour"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

type CreatePresetTourResponse struct {
	PresetTourToken onvif.ReferenceToken
}

type ModifyPresetTour struct {
	XMLName      string               `xml:"tptz:ModifyPresetTour"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	PresetTour   onvif.PresetTour     `xml:"tptz:PresetTour"`
}

type ModifyPresetTourResponse struct {
}

type OperatePresetTour struct {
	XMLName         string                       `xml:"tptz:OperatePresetTour"`
	ProfileToken    onvif.ReferenceToken         `xml:"tptz:ProfileToken"`
	PresetTourToken onvif.ReferenceToken         `xml:"tptz:PresetTourToken"`
	Operation       onvif.PTZPresetTourOperation `xml:"tptz:Operation"`
}

type OperatePresetTourResponse struct {
}

type RemovePresetTour struct {
	XMLName         string               `xml:"tptz:RemovePresetTour"`
	ProfileToken    onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	PresetTourToken onvif.ReferenceToken `xml:"tptz:PresetTourToken"`
}

type RemovePresetTourResponse struct {
}

type GetCompatibleConfigurations struct {
	XMLName      string               `xml:"tptz:GetCompatibleConfigurations"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

type GetCompatibleConfigurationsResponse struct {
	PTZConfiguration onvif.PTZConfiguration
}

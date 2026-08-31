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

// Code generated from profiles/S.profile : DO NOT EDIT.

package sdk

import (
	"context"

	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/event"
	"github.com/jfsmig/onvif/media"
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/ptz"
)

// ProfileS offers the operations that the ONVIF Profile S specification lists for
// a client, over one connection to one appliance.
//
// It is not a conformance statement. This library is not certified, the ONVIF conformance
// test tool is not run against it, and an appliance advertising the required services may
// still reject any of these operations. Membership is curated by hand from the
// specification, one cited section per line, in profiles/S.profile.
//
// Not to be confused with an ONVIF *media* profile, which is a per-stream configuration on
// the appliance and is reached through the media service.
type ProfileS struct {
	client *networking.Client
}

// NewProfileS builds the client and reports whether the appliance advertises the services
// that Profile S requires unconditionally (device, media).
//
// It reports on the endpoints already learnt when the appliance was loaded, so it sends
// nothing and a true result means "worth trying", not "will succeed".
//
// The remaining services are conditional in this Profile (event, ptz), so their absence
// does not make the client unusable: their operations return utils.ErrNoService, and the
// Has* predicates below let a caller check first.
func NewProfileS(client *networking.Client) (*ProfileS, bool) {
	if _, ok := client.HasEndpoint("device"); !ok {
		return nil, false
	}
	if _, ok := client.HasEndpoint("media"); !ok {
		return nil, false
	}
	return &ProfileS{client: client}, true
}

// HasEvent reports whether the appliance advertises the event service, which Profile
// S treats as conditional.
func (p *ProfileS) HasEvent() bool {
	_, ok := p.client.HasEndpoint("event")
	return ok
}

// HasPTZ reports whether the appliance advertises the ptz service, which Profile
// S treats as conditional.
func (p *ProfileS) HasPTZ() bool {
	_, ok := p.client.HasEndpoint("ptz")
	return ok
}

// AbsoluteMove performs the ONVIF ptz operation AbsoluteMove.
//
// Profile S 8.4 PTZ – Absolute Positioning, mandatory for clients.
func (p *ProfileS) AbsoluteMove(ctx context.Context, request ptz.AbsoluteMove) (ptz.AbsoluteMoveResponse, error) {
	return ptz.Call_AbsoluteMove(ctx, p.client, request)
}

// AddAudioEncoderConfiguration performs the ONVIF media operation AddAudioEncoderConfiguration.
//
// Profile S 8.9 Audio Streaming, mandatory for clients.
func (p *ProfileS) AddAudioEncoderConfiguration(ctx context.Context, request media.AddAudioEncoderConfiguration) (media.AddAudioEncoderConfigurationResponse, error) {
	return media.Call_AddAudioEncoderConfiguration(ctx, p.client, request)
}

// AddAudioSourceConfiguration performs the ONVIF media operation AddAudioSourceConfiguration.
//
// Profile S 8.9 Audio Streaming, mandatory for clients.
func (p *ProfileS) AddAudioSourceConfiguration(ctx context.Context, request media.AddAudioSourceConfiguration) (media.AddAudioSourceConfigurationResponse, error) {
	return media.Call_AddAudioSourceConfiguration(ctx, p.client, request)
}

// AddIPAddressFilter performs the ONVIF device operation AddIPAddressFilter.
//
// Profile S 8.17 IP Address Filtering, mandatory for clients.
func (p *ProfileS) AddIPAddressFilter(ctx context.Context, request device.AddIPAddressFilter) (device.AddIPAddressFilterResponse, error) {
	return device.Call_AddIPAddressFilter(ctx, p.client, request)
}

// AddMetadataConfiguration performs the ONVIF media operation AddMetadataConfiguration.
//
// Profile S 7.13 Metadata Configuration, optional.
func (p *ProfileS) AddMetadataConfiguration(ctx context.Context, request media.AddMetadataConfiguration) (media.AddMetadataConfigurationResponse, error) {
	return media.Call_AddMetadataConfiguration(ctx, p.client, request)
}

// AddPTZConfiguration performs the ONVIF media operation AddPTZConfiguration.
//
// Profile S 8.3 PTZ, mandatory for clients.
func (p *ProfileS) AddPTZConfiguration(ctx context.Context, request media.AddPTZConfiguration) (media.AddPTZConfigurationResponse, error) {
	return media.Call_AddPTZConfiguration(ctx, p.client, request)
}

// AddScopes performs the ONVIF device operation AddScopes.
//
// Profile S 7.3 Discovery, optional.
func (p *ProfileS) AddScopes(ctx context.Context, request device.AddScopes) (device.AddScopesResponse, error) {
	return device.Call_AddScopes(ctx, p.client, request)
}

// AddVideoEncoderConfiguration performs the ONVIF media operation AddVideoEncoderConfiguration.
//
// Profile S 7.10 Video Encoder Configuration, optional.
func (p *ProfileS) AddVideoEncoderConfiguration(ctx context.Context, request media.AddVideoEncoderConfiguration) (media.AddVideoEncoderConfigurationResponse, error) {
	return media.Call_AddVideoEncoderConfiguration(ctx, p.client, request)
}

// AddVideoSourceConfiguration performs the ONVIF media operation AddVideoSourceConfiguration.
//
// Profile S 7.12 Video Source Configuration, mandatory for clients.
func (p *ProfileS) AddVideoSourceConfiguration(ctx context.Context, request media.AddVideoSourceConfiguration) (media.AddVideoSourceConfigurationResponse, error) {
	return media.Call_AddVideoSourceConfiguration(ctx, p.client, request)
}

// ContinuousMove performs the ONVIF ptz operation ContinuousMove.
//
// Profile S 8.3 PTZ, mandatory for clients.
func (p *ProfileS) ContinuousMove(ctx context.Context, request ptz.ContinuousMove) (ptz.ContinuousMoveResponse, error) {
	return ptz.Call_ContinuousMove(ctx, p.client, request)
}

// CreateProfile performs the ONVIF media operation CreateProfile.
//
// Profile S 7.11 Media Profile Configuration, mandatory for clients.
func (p *ProfileS) CreateProfile(ctx context.Context, request media.CreateProfile) (media.CreateProfileResponse, error) {
	return media.Call_CreateProfile(ctx, p.client, request)
}

// CreatePullPointSubscription performs the ONVIF event operation CreatePullPointSubscription.
//
// Profile S 7.7 Event Handling, mandatory for clients, subject to the footnote of that section.
func (p *ProfileS) CreatePullPointSubscription(ctx context.Context, request event.CreatePullPointSubscription) (event.CreatePullPointSubscriptionResponse, error) {
	return event.Call_CreatePullPointSubscription(ctx, p.client, request)
}

// CreateUsers performs the ONVIF device operation CreateUsers.
//
// Profile S 7.6 User Handling, mandatory for clients.
func (p *ProfileS) CreateUsers(ctx context.Context, request device.CreateUsers) (device.CreateUsersResponse, error) {
	return device.Call_CreateUsers(ctx, p.client, request)
}

// DeleteProfile performs the ONVIF media operation DeleteProfile.
//
// Profile S 7.11 Media Profile Configuration, optional.
func (p *ProfileS) DeleteProfile(ctx context.Context, request media.DeleteProfile) (media.DeleteProfileResponse, error) {
	return media.Call_DeleteProfile(ctx, p.client, request)
}

// DeleteUsers performs the ONVIF device operation DeleteUsers.
//
// Profile S 7.6 User Handling, mandatory for clients.
func (p *ProfileS) DeleteUsers(ctx context.Context, request device.DeleteUsers) (device.DeleteUsersResponse, error) {
	return device.Call_DeleteUsers(ctx, p.client, request)
}

// GetAudioEncoderConfiguration performs the ONVIF media operation GetAudioEncoderConfiguration.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) GetAudioEncoderConfiguration(ctx context.Context, request media.GetAudioEncoderConfiguration) (media.GetAudioEncoderConfigurationResponse, error) {
	return media.Call_GetAudioEncoderConfiguration(ctx, p.client, request)
}

// GetAudioEncoderConfigurationOptions performs the ONVIF media operation GetAudioEncoderConfigurationOptions.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) GetAudioEncoderConfigurationOptions(ctx context.Context, request media.GetAudioEncoderConfigurationOptions) (media.GetAudioEncoderConfigurationOptionsResponse, error) {
	return media.Call_GetAudioEncoderConfigurationOptions(ctx, p.client, request)
}

// GetAudioEncoderConfigurations performs the ONVIF media operation GetAudioEncoderConfigurations.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) GetAudioEncoderConfigurations(ctx context.Context, request media.GetAudioEncoderConfigurations) (media.GetAudioEncoderConfigurationsResponse, error) {
	return media.Call_GetAudioEncoderConfigurations(ctx, p.client, request)
}

// GetAudioSourceConfiguration performs the ONVIF media operation GetAudioSourceConfiguration.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) GetAudioSourceConfiguration(ctx context.Context, request media.GetAudioSourceConfiguration) (media.GetAudioSourceConfigurationResponse, error) {
	return media.Call_GetAudioSourceConfiguration(ctx, p.client, request)
}

// GetAudioSourceConfigurationOptions performs the ONVIF media operation GetAudioSourceConfigurationOptions.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) GetAudioSourceConfigurationOptions(ctx context.Context, request media.GetAudioSourceConfigurationOptions) (media.GetAudioSourceConfigurationOptionsResponse, error) {
	return media.Call_GetAudioSourceConfigurationOptions(ctx, p.client, request)
}

// GetAudioSourceConfigurations performs the ONVIF media operation GetAudioSourceConfigurations.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) GetAudioSourceConfigurations(ctx context.Context, request media.GetAudioSourceConfigurations) (media.GetAudioSourceConfigurationsResponse, error) {
	return media.Call_GetAudioSourceConfigurations(ctx, p.client, request)
}

// GetAudioSources performs the ONVIF media operation GetAudioSources.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) GetAudioSources(ctx context.Context, request media.GetAudioSources) (media.GetAudioSourcesResponse, error) {
	return media.Call_GetAudioSources(ctx, p.client, request)
}

// GetCapabilities performs the ONVIF device operation GetCapabilities.
//
// Profile S 7.2 Capabilities, mandatory for clients.
func (p *ProfileS) GetCapabilities(ctx context.Context, request device.GetCapabilities) (device.GetCapabilitiesResponse, error) {
	return device.Call_GetCapabilities(ctx, p.client, request)
}

// GetCompatibleAudioEncoderConfigurations performs the ONVIF media operation GetCompatibleAudioEncoderConfigurations.
//
// Profile S 8.9 Audio Streaming, mandatory for clients.
func (p *ProfileS) GetCompatibleAudioEncoderConfigurations(ctx context.Context, request media.GetCompatibleAudioEncoderConfigurations) (media.GetCompatibleAudioEncoderConfigurationsResponse, error) {
	return media.Call_GetCompatibleAudioEncoderConfigurations(ctx, p.client, request)
}

// GetCompatibleAudioSourceConfigurations performs the ONVIF media operation GetCompatibleAudioSourceConfigurations.
//
// Profile S 8.9 Audio Streaming, mandatory for clients.
func (p *ProfileS) GetCompatibleAudioSourceConfigurations(ctx context.Context, request media.GetCompatibleAudioSourceConfigurations) (media.GetCompatibleAudioSourceConfigurationsResponse, error) {
	return media.Call_GetCompatibleAudioSourceConfigurations(ctx, p.client, request)
}

// GetCompatibleMetadataConfigurations performs the ONVIF media operation GetCompatibleMetadataConfigurations.
//
// Profile S 7.13 Metadata Configuration, optional.
func (p *ProfileS) GetCompatibleMetadataConfigurations(ctx context.Context, request media.GetCompatibleMetadataConfigurations) (media.GetCompatibleMetadataConfigurationsResponse, error) {
	return media.Call_GetCompatibleMetadataConfigurations(ctx, p.client, request)
}

// GetCompatibleVideoEncoderConfigurations performs the ONVIF media operation GetCompatibleVideoEncoderConfigurations.
//
// Profile S 7.10 Video Encoder Configuration, optional.
func (p *ProfileS) GetCompatibleVideoEncoderConfigurations(ctx context.Context, request media.GetCompatibleVideoEncoderConfigurations) (media.GetCompatibleVideoEncoderConfigurationsResponse, error) {
	return media.Call_GetCompatibleVideoEncoderConfigurations(ctx, p.client, request)
}

// GetCompatibleVideoSourceConfigurations performs the ONVIF media operation GetCompatibleVideoSourceConfigurations.
//
// Profile S 7.12 Video Source Configuration, mandatory for clients.
func (p *ProfileS) GetCompatibleVideoSourceConfigurations(ctx context.Context, request media.GetCompatibleVideoSourceConfigurations) (media.GetCompatibleVideoSourceConfigurationsResponse, error) {
	return media.Call_GetCompatibleVideoSourceConfigurations(ctx, p.client, request)
}

// GetConfiguration performs the ONVIF ptz operation GetConfiguration.
//
// Profile S 8.3 PTZ, optional.
func (p *ProfileS) GetConfiguration(ctx context.Context, request ptz.GetConfiguration) (ptz.GetConfigurationResponse, error) {
	return ptz.Call_GetConfiguration(ctx, p.client, request)
}

// GetConfigurationOptions performs the ONVIF ptz operation GetConfigurationOptions.
//
// Profile S 8.3 PTZ, optional.
func (p *ProfileS) GetConfigurationOptions(ctx context.Context, request ptz.GetConfigurationOptions) (ptz.GetConfigurationOptionsResponse, error) {
	return ptz.Call_GetConfigurationOptions(ctx, p.client, request)
}

// GetConfigurations performs the ONVIF ptz operation GetConfigurations.
//
// Profile S 8.3 PTZ, mandatory for clients.
func (p *ProfileS) GetConfigurations(ctx context.Context, request ptz.GetConfigurations) (ptz.GetConfigurationsResponse, error) {
	return ptz.Call_GetConfigurations(ctx, p.client, request)
}

// GetDNS performs the ONVIF device operation GetDNS.
//
// Profile S 7.4 Network Configuration, optional.
func (p *ProfileS) GetDNS(ctx context.Context, request device.GetDNS) (device.GetDNSResponse, error) {
	return device.Call_GetDNS(ctx, p.client, request)
}

// GetDeviceInformation performs the ONVIF device operation GetDeviceInformation.
//
// Profile S 7.5 System, mandatory for clients.
func (p *ProfileS) GetDeviceInformation(ctx context.Context, request device.GetDeviceInformation) (device.GetDeviceInformationResponse, error) {
	return device.Call_GetDeviceInformation(ctx, p.client, request)
}

// GetDiscoveryMode performs the ONVIF device operation GetDiscoveryMode.
//
// Profile S 7.3 Discovery, optional.
func (p *ProfileS) GetDiscoveryMode(ctx context.Context, request device.GetDiscoveryMode) (device.GetDiscoveryModeResponse, error) {
	return device.Call_GetDiscoveryMode(ctx, p.client, request)
}

// GetDynamicDNS performs the ONVIF device operation GetDynamicDNS.
//
// Profile S 8.15 Dynamic DNS, mandatory for clients.
func (p *ProfileS) GetDynamicDNS(ctx context.Context, request device.GetDynamicDNS) (device.GetDynamicDNSResponse, error) {
	return device.Call_GetDynamicDNS(ctx, p.client, request)
}

// GetEventProperties performs the ONVIF event operation GetEventProperties.
//
// Profile S 7.7 Event Handling, optional.
func (p *ProfileS) GetEventProperties(ctx context.Context, request event.GetEventProperties) (event.GetEventPropertiesResponse, error) {
	return event.Call_GetEventProperties(ctx, p.client, request)
}

// GetGuaranteedNumberOfVideoEncoderInstances performs the ONVIF media operation GetGuaranteedNumberOfVideoEncoderInstances.
//
// Profile S 7.10 Video Encoder Configuration, optional.
func (p *ProfileS) GetGuaranteedNumberOfVideoEncoderInstances(ctx context.Context, request media.GetGuaranteedNumberOfVideoEncoderInstances) (media.GetGuaranteedNumberOfVideoEncoderInstancesResponse, error) {
	return media.Call_GetGuaranteedNumberOfVideoEncoderInstances(ctx, p.client, request)
}

// GetHostname performs the ONVIF device operation GetHostname.
//
// Profile S 7.4 Network Configuration, optional.
func (p *ProfileS) GetHostname(ctx context.Context, request device.GetHostname) (device.GetHostnameResponse, error) {
	return device.Call_GetHostname(ctx, p.client, request)
}

// GetIPAddressFilter performs the ONVIF device operation GetIPAddressFilter.
//
// Profile S 8.17 IP Address Filtering, mandatory for clients.
func (p *ProfileS) GetIPAddressFilter(ctx context.Context, request device.GetIPAddressFilter) (device.GetIPAddressFilterResponse, error) {
	return device.Call_GetIPAddressFilter(ctx, p.client, request)
}

// GetMetadataConfiguration performs the ONVIF media operation GetMetadataConfiguration.
//
// Profile S 7.13 Metadata Configuration, optional.
func (p *ProfileS) GetMetadataConfiguration(ctx context.Context, request media.GetMetadataConfiguration) (media.GetMetadataConfigurationResponse, error) {
	return media.Call_GetMetadataConfiguration(ctx, p.client, request)
}

// GetMetadataConfigurationOptions performs the ONVIF media operation GetMetadataConfigurationOptions.
//
// Profile S 7.13 Metadata Configuration, mandatory for clients.
func (p *ProfileS) GetMetadataConfigurationOptions(ctx context.Context, request media.GetMetadataConfigurationOptions) (media.GetMetadataConfigurationOptionsResponse, error) {
	return media.Call_GetMetadataConfigurationOptions(ctx, p.client, request)
}

// GetMetadataConfigurations performs the ONVIF media operation GetMetadataConfigurations.
//
// Profile S 7.13 Metadata Configuration, optional.
func (p *ProfileS) GetMetadataConfigurations(ctx context.Context, request media.GetMetadataConfigurations) (media.GetMetadataConfigurationsResponse, error) {
	return media.Call_GetMetadataConfigurations(ctx, p.client, request)
}

// GetNTP performs the ONVIF device operation GetNTP.
//
// Profile S 8.14 NTP, mandatory for clients.
func (p *ProfileS) GetNTP(ctx context.Context, request device.GetNTP) (device.GetNTPResponse, error) {
	return device.Call_GetNTP(ctx, p.client, request)
}

// GetNetworkDefaultGateway performs the ONVIF device operation GetNetworkDefaultGateway.
//
// Profile S 7.4 Network Configuration, mandatory for clients.
func (p *ProfileS) GetNetworkDefaultGateway(ctx context.Context, request device.GetNetworkDefaultGateway) (device.GetNetworkDefaultGatewayResponse, error) {
	return device.Call_GetNetworkDefaultGateway(ctx, p.client, request)
}

// GetNetworkInterfaces performs the ONVIF device operation GetNetworkInterfaces.
//
// Profile S 7.4 Network Configuration, mandatory for clients.
func (p *ProfileS) GetNetworkInterfaces(ctx context.Context, request device.GetNetworkInterfaces) (device.GetNetworkInterfacesResponse, error) {
	return device.Call_GetNetworkInterfaces(ctx, p.client, request)
}

// GetNetworkProtocols performs the ONVIF device operation GetNetworkProtocols.
//
// Profile S 7.4 Network Configuration, optional.
func (p *ProfileS) GetNetworkProtocols(ctx context.Context, request device.GetNetworkProtocols) (device.GetNetworkProtocolsResponse, error) {
	return device.Call_GetNetworkProtocols(ctx, p.client, request)
}

// GetNode performs the ONVIF ptz operation GetNode.
//
// Profile S 8.3 PTZ, mandatory for clients.
func (p *ProfileS) GetNode(ctx context.Context, request ptz.GetNode) (ptz.GetNodeResponse, error) {
	return ptz.Call_GetNode(ctx, p.client, request)
}

// GetNodes performs the ONVIF ptz operation GetNodes.
//
// Profile S 8.3 PTZ, mandatory for clients.
func (p *ProfileS) GetNodes(ctx context.Context, request ptz.GetNodes) (ptz.GetNodesResponse, error) {
	return ptz.Call_GetNodes(ctx, p.client, request)
}

// GetPresets performs the ONVIF ptz operation GetPresets.
//
// Profile S 8.6 PTZ – Presets, mandatory for clients.
func (p *ProfileS) GetPresets(ctx context.Context, request ptz.GetPresets) (ptz.GetPresetsResponse, error) {
	return ptz.Call_GetPresets(ctx, p.client, request)
}

// GetProfile performs the ONVIF media operation GetProfile.
//
// Profile S 7.11 Media Profile Configuration, optional.
func (p *ProfileS) GetProfile(ctx context.Context, request media.GetProfile) (media.GetProfileResponse, error) {
	return media.Call_GetProfile(ctx, p.client, request)
}

// GetProfiles performs the ONVIF media operation GetProfiles.
//
// Profile S 7.8 Video Streaming, mandatory for clients.
func (p *ProfileS) GetProfiles(ctx context.Context, request media.GetProfiles) (media.GetProfilesResponse, error) {
	return media.Call_GetProfiles(ctx, p.client, request)
}

// GetRelayOutputs performs the ONVIF device operation GetRelayOutputs.
//
// Profile S 8.13 Relay Outputs, mandatory for clients.
func (p *ProfileS) GetRelayOutputs(ctx context.Context, request device.GetRelayOutputs) (device.GetRelayOutputsResponse, error) {
	return device.Call_GetRelayOutputs(ctx, p.client, request)
}

// GetScopes performs the ONVIF device operation GetScopes.
//
// Profile S 7.3 Discovery, optional.
func (p *ProfileS) GetScopes(ctx context.Context, request device.GetScopes) (device.GetScopesResponse, error) {
	return device.Call_GetScopes(ctx, p.client, request)
}

// GetStatus performs the ONVIF ptz operation GetStatus.
//
// Profile S 8.3 PTZ, optional.
func (p *ProfileS) GetStatus(ctx context.Context, request ptz.GetStatus) (ptz.GetStatusResponse, error) {
	return ptz.Call_GetStatus(ctx, p.client, request)
}

// GetStreamUri performs the ONVIF media operation GetStreamUri.
//
// Profile S 7.8 Video Streaming, mandatory for clients.
func (p *ProfileS) GetStreamUri(ctx context.Context, request media.GetStreamUri) (media.GetStreamUriResponse, error) {
	return media.Call_GetStreamUri(ctx, p.client, request)
}

// GetSystemDateAndTime performs the ONVIF device operation GetSystemDateAndTime.
//
// Profile S 7.5 System, optional.
func (p *ProfileS) GetSystemDateAndTime(ctx context.Context, request device.GetSystemDateAndTime) (device.GetSystemDateAndTimeResponse, error) {
	return device.Call_GetSystemDateAndTime(ctx, p.client, request)
}

// GetUsers performs the ONVIF device operation GetUsers.
//
// Profile S 7.6 User Handling, mandatory for clients.
func (p *ProfileS) GetUsers(ctx context.Context, request device.GetUsers) (device.GetUsersResponse, error) {
	return device.Call_GetUsers(ctx, p.client, request)
}

// GetVideoEncoderConfiguration performs the ONVIF media operation GetVideoEncoderConfiguration.
//
// Profile S 7.10 Video Encoder Configuration, optional.
func (p *ProfileS) GetVideoEncoderConfiguration(ctx context.Context, request media.GetVideoEncoderConfiguration) (media.GetVideoEncoderConfigurationResponse, error) {
	return media.Call_GetVideoEncoderConfiguration(ctx, p.client, request)
}

// GetVideoEncoderConfigurationOptions performs the ONVIF media operation GetVideoEncoderConfigurationOptions.
//
// Profile S 7.10 Video Encoder Configuration, mandatory for clients.
func (p *ProfileS) GetVideoEncoderConfigurationOptions(ctx context.Context, request media.GetVideoEncoderConfigurationOptions) (media.GetVideoEncoderConfigurationOptionsResponse, error) {
	return media.Call_GetVideoEncoderConfigurationOptions(ctx, p.client, request)
}

// GetVideoEncoderConfigurations performs the ONVIF media operation GetVideoEncoderConfigurations.
//
// Profile S 7.10 Video Encoder Configuration, optional.
func (p *ProfileS) GetVideoEncoderConfigurations(ctx context.Context, request media.GetVideoEncoderConfigurations) (media.GetVideoEncoderConfigurationsResponse, error) {
	return media.Call_GetVideoEncoderConfigurations(ctx, p.client, request)
}

// GetVideoSourceConfiguration performs the ONVIF media operation GetVideoSourceConfiguration.
//
// Profile S 7.12 Video Source Configuration, optional.
func (p *ProfileS) GetVideoSourceConfiguration(ctx context.Context, request media.GetVideoSourceConfiguration) (media.GetVideoSourceConfigurationResponse, error) {
	return media.Call_GetVideoSourceConfiguration(ctx, p.client, request)
}

// GetVideoSourceConfigurationOptions performs the ONVIF media operation GetVideoSourceConfigurationOptions.
//
// Profile S 7.12 Video Source Configuration, mandatory for clients.
func (p *ProfileS) GetVideoSourceConfigurationOptions(ctx context.Context, request media.GetVideoSourceConfigurationOptions) (media.GetVideoSourceConfigurationOptionsResponse, error) {
	return media.Call_GetVideoSourceConfigurationOptions(ctx, p.client, request)
}

// GetVideoSourceConfigurations performs the ONVIF media operation GetVideoSourceConfigurations.
//
// Profile S 7.12 Video Source Configuration, optional.
func (p *ProfileS) GetVideoSourceConfigurations(ctx context.Context, request media.GetVideoSourceConfigurations) (media.GetVideoSourceConfigurationsResponse, error) {
	return media.Call_GetVideoSourceConfigurations(ctx, p.client, request)
}

// GetVideoSources performs the ONVIF media operation GetVideoSources.
//
// Profile S 7.12 Video Source Configuration, optional.
func (p *ProfileS) GetVideoSources(ctx context.Context, request media.GetVideoSources) (media.GetVideoSourcesResponse, error) {
	return media.Call_GetVideoSources(ctx, p.client, request)
}

// GetWsdlUrl performs the ONVIF device operation GetWsdlUrl.
//
// Profile S 7.2 Capabilities, optional.
func (p *ProfileS) GetWsdlUrl(ctx context.Context, request device.GetWsdlUrl) (device.GetWsdlUrlResponse, error) {
	return device.Call_GetWsdlUrl(ctx, p.client, request)
}

// GetZeroConfiguration performs the ONVIF device operation GetZeroConfiguration.
//
// Profile S 8.16 Zero Configuration, mandatory for clients.
func (p *ProfileS) GetZeroConfiguration(ctx context.Context, request device.GetZeroConfiguration) (device.GetZeroConfigurationResponse, error) {
	return device.Call_GetZeroConfiguration(ctx, p.client, request)
}

// GotoHomePosition performs the ONVIF ptz operation GotoHomePosition.
//
// Profile S 8.7 PTZ – Home Position, mandatory for clients.
func (p *ProfileS) GotoHomePosition(ctx context.Context, request ptz.GotoHomePosition) (ptz.GotoHomePositionResponse, error) {
	return ptz.Call_GotoHomePosition(ctx, p.client, request)
}

// GotoPreset performs the ONVIF ptz operation GotoPreset.
//
// Profile S 8.6 PTZ – Presets, mandatory for clients.
func (p *ProfileS) GotoPreset(ctx context.Context, request ptz.GotoPreset) (ptz.GotoPresetResponse, error) {
	return ptz.Call_GotoPreset(ctx, p.client, request)
}

// PullMessages performs the ONVIF event operation PullMessages.
//
// Profile S 7.7 Event Handling, mandatory for clients, subject to the footnote of that section.
func (p *ProfileS) PullMessages(ctx context.Context, request event.PullMessages) (event.PullMessagesResponse, error) {
	return event.Call_PullMessages(ctx, p.client, request)
}

// RelativeMove performs the ONVIF ptz operation RelativeMove.
//
// Profile S 8.5 PTZ – Relative Positioning, mandatory for clients.
func (p *ProfileS) RelativeMove(ctx context.Context, request ptz.RelativeMove) (ptz.RelativeMoveResponse, error) {
	return ptz.Call_RelativeMove(ctx, p.client, request)
}

// RemoveAudioEncoderConfiguration performs the ONVIF media operation RemoveAudioEncoderConfiguration.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) RemoveAudioEncoderConfiguration(ctx context.Context, request media.RemoveAudioEncoderConfiguration) (media.RemoveAudioEncoderConfigurationResponse, error) {
	return media.Call_RemoveAudioEncoderConfiguration(ctx, p.client, request)
}

// RemoveAudioSourceConfiguration performs the ONVIF media operation RemoveAudioSourceConfiguration.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) RemoveAudioSourceConfiguration(ctx context.Context, request media.RemoveAudioSourceConfiguration) (media.RemoveAudioSourceConfigurationResponse, error) {
	return media.Call_RemoveAudioSourceConfiguration(ctx, p.client, request)
}

// RemoveIPAddressFilter performs the ONVIF device operation RemoveIPAddressFilter.
//
// Profile S 8.17 IP Address Filtering, mandatory for clients.
func (p *ProfileS) RemoveIPAddressFilter(ctx context.Context, request device.RemoveIPAddressFilter) (device.RemoveIPAddressFilterResponse, error) {
	return device.Call_RemoveIPAddressFilter(ctx, p.client, request)
}

// RemoveMetadataConfiguration performs the ONVIF media operation RemoveMetadataConfiguration.
//
// Profile S 7.13 Metadata Configuration, optional.
func (p *ProfileS) RemoveMetadataConfiguration(ctx context.Context, request media.RemoveMetadataConfiguration) (media.RemoveMetadataConfigurationResponse, error) {
	return media.Call_RemoveMetadataConfiguration(ctx, p.client, request)
}

// RemovePTZConfiguration performs the ONVIF media operation RemovePTZConfiguration.
//
// Profile S 8.3 PTZ, optional.
func (p *ProfileS) RemovePTZConfiguration(ctx context.Context, request media.RemovePTZConfiguration) (media.RemovePTZConfigurationResponse, error) {
	return media.Call_RemovePTZConfiguration(ctx, p.client, request)
}

// RemovePreset performs the ONVIF ptz operation RemovePreset.
//
// Profile S 8.6 PTZ – Presets, optional.
func (p *ProfileS) RemovePreset(ctx context.Context, request ptz.RemovePreset) (ptz.RemovePresetResponse, error) {
	return ptz.Call_RemovePreset(ctx, p.client, request)
}

// RemoveScopes performs the ONVIF device operation RemoveScopes.
//
// Profile S 7.3 Discovery, optional.
func (p *ProfileS) RemoveScopes(ctx context.Context, request device.RemoveScopes) (device.RemoveScopesResponse, error) {
	return device.Call_RemoveScopes(ctx, p.client, request)
}

// RemoveVideoEncoderConfiguration performs the ONVIF media operation RemoveVideoEncoderConfiguration.
//
// Profile S 7.10 Video Encoder Configuration, optional.
func (p *ProfileS) RemoveVideoEncoderConfiguration(ctx context.Context, request media.RemoveVideoEncoderConfiguration) (media.RemoveVideoEncoderConfigurationResponse, error) {
	return media.Call_RemoveVideoEncoderConfiguration(ctx, p.client, request)
}

// RemoveVideoSourceConfiguration performs the ONVIF media operation RemoveVideoSourceConfiguration.
//
// Profile S 7.12 Video Source Configuration, optional.
func (p *ProfileS) RemoveVideoSourceConfiguration(ctx context.Context, request media.RemoveVideoSourceConfiguration) (media.RemoveVideoSourceConfigurationResponse, error) {
	return media.Call_RemoveVideoSourceConfiguration(ctx, p.client, request)
}

// Renew performs the ONVIF event operation Renew.
//
// Profile S 7.7 Event Handling, optional.
func (p *ProfileS) Renew(ctx context.Context, request event.Renew) (event.RenewResponse, error) {
	return event.Call_Renew(ctx, p.client, request)
}

// SendAuxiliaryCommandPTZ performs the ONVIF ptz operation SendAuxiliaryCommand.
//
// Profile S 8.8 PTZ – Auxiliary Command Position, mandatory for clients.
func (p *ProfileS) SendAuxiliaryCommandPTZ(ctx context.Context, request ptz.SendAuxiliaryCommand) (ptz.SendAuxiliaryCommandResponse, error) {
	return ptz.Call_SendAuxiliaryCommand(ctx, p.client, request)
}

// SetAudioEncoderConfiguration performs the ONVIF media operation SetAudioEncoderConfiguration.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) SetAudioEncoderConfiguration(ctx context.Context, request media.SetAudioEncoderConfiguration) (media.SetAudioEncoderConfigurationResponse, error) {
	return media.Call_SetAudioEncoderConfiguration(ctx, p.client, request)
}

// SetAudioSourceConfiguration performs the ONVIF media operation SetAudioSourceConfiguration.
//
// Profile S 8.9 Audio Streaming, optional.
func (p *ProfileS) SetAudioSourceConfiguration(ctx context.Context, request media.SetAudioSourceConfiguration) (media.SetAudioSourceConfigurationResponse, error) {
	return media.Call_SetAudioSourceConfiguration(ctx, p.client, request)
}

// SetConfiguration performs the ONVIF ptz operation SetConfiguration.
//
// Profile S 8.3 PTZ, optional.
func (p *ProfileS) SetConfiguration(ctx context.Context, request ptz.SetConfiguration) (ptz.SetConfigurationResponse, error) {
	return ptz.Call_SetConfiguration(ctx, p.client, request)
}

// SetDNS performs the ONVIF device operation SetDNS.
//
// Profile S 7.4 Network Configuration, optional.
func (p *ProfileS) SetDNS(ctx context.Context, request device.SetDNS) (device.SetDNSResponse, error) {
	return device.Call_SetDNS(ctx, p.client, request)
}

// SetDiscoveryMode performs the ONVIF device operation SetDiscoveryMode.
//
// Profile S 7.3 Discovery, optional.
func (p *ProfileS) SetDiscoveryMode(ctx context.Context, request device.SetDiscoveryMode) (device.SetDiscoveryModeResponse, error) {
	return device.Call_SetDiscoveryMode(ctx, p.client, request)
}

// SetDynamicDNS performs the ONVIF device operation SetDynamicDNS.
//
// Profile S 8.15 Dynamic DNS, mandatory for clients.
func (p *ProfileS) SetDynamicDNS(ctx context.Context, request device.SetDynamicDNS) (device.SetDynamicDNSResponse, error) {
	return device.Call_SetDynamicDNS(ctx, p.client, request)
}

// SetHomePosition performs the ONVIF ptz operation SetHomePosition.
//
// Profile S 8.7 PTZ – Home Position, optional.
func (p *ProfileS) SetHomePosition(ctx context.Context, request ptz.SetHomePosition) (ptz.SetHomePositionResponse, error) {
	return ptz.Call_SetHomePosition(ctx, p.client, request)
}

// SetHostname performs the ONVIF device operation SetHostname.
//
// Profile S 7.4 Network Configuration, optional.
func (p *ProfileS) SetHostname(ctx context.Context, request device.SetHostname) (device.SetHostnameResponse, error) {
	return device.Call_SetHostname(ctx, p.client, request)
}

// SetIPAddressFilter performs the ONVIF device operation SetIPAddressFilter.
//
// Profile S 8.17 IP Address Filtering, mandatory for clients.
func (p *ProfileS) SetIPAddressFilter(ctx context.Context, request device.SetIPAddressFilter) (device.SetIPAddressFilterResponse, error) {
	return device.Call_SetIPAddressFilter(ctx, p.client, request)
}

// SetMetadataConfiguration performs the ONVIF media operation SetMetadataConfiguration.
//
// Profile S 7.13 Metadata Configuration, mandatory for clients.
func (p *ProfileS) SetMetadataConfiguration(ctx context.Context, request media.SetMetadataConfiguration) (media.SetMetadataConfigurationResponse, error) {
	return media.Call_SetMetadataConfiguration(ctx, p.client, request)
}

// SetNTP performs the ONVIF device operation SetNTP.
//
// Profile S 8.14 NTP, mandatory for clients.
func (p *ProfileS) SetNTP(ctx context.Context, request device.SetNTP) (device.SetNTPResponse, error) {
	return device.Call_SetNTP(ctx, p.client, request)
}

// SetNetworkDefaultGateway performs the ONVIF device operation SetNetworkDefaultGateway.
//
// Profile S 7.4 Network Configuration, mandatory for clients.
func (p *ProfileS) SetNetworkDefaultGateway(ctx context.Context, request device.SetNetworkDefaultGateway) (device.SetNetworkDefaultGatewayResponse, error) {
	return device.Call_SetNetworkDefaultGateway(ctx, p.client, request)
}

// SetNetworkInterfaces performs the ONVIF device operation SetNetworkInterfaces.
//
// Profile S 7.4 Network Configuration, mandatory for clients.
func (p *ProfileS) SetNetworkInterfaces(ctx context.Context, request device.SetNetworkInterfaces) (device.SetNetworkInterfacesResponse, error) {
	return device.Call_SetNetworkInterfaces(ctx, p.client, request)
}

// SetNetworkProtocols performs the ONVIF device operation SetNetworkProtocols.
//
// Profile S 7.4 Network Configuration, optional.
func (p *ProfileS) SetNetworkProtocols(ctx context.Context, request device.SetNetworkProtocols) (device.SetNetworkProtocolsResponse, error) {
	return device.Call_SetNetworkProtocols(ctx, p.client, request)
}

// SetPreset performs the ONVIF ptz operation SetPreset.
//
// Profile S 8.6 PTZ – Presets, optional.
func (p *ProfileS) SetPreset(ctx context.Context, request ptz.SetPreset) (ptz.SetPresetResponse, error) {
	return ptz.Call_SetPreset(ctx, p.client, request)
}

// SetRelayOutputSettings performs the ONVIF device operation SetRelayOutputSettings.
//
// Profile S 8.13 Relay Outputs, mandatory for clients.
func (p *ProfileS) SetRelayOutputSettings(ctx context.Context, request device.SetRelayOutputSettings) (device.SetRelayOutputSettingsResponse, error) {
	return device.Call_SetRelayOutputSettings(ctx, p.client, request)
}

// SetRelayOutputState performs the ONVIF device operation SetRelayOutputState.
//
// Profile S 8.13 Relay Outputs, mandatory for clients.
func (p *ProfileS) SetRelayOutputState(ctx context.Context, request device.SetRelayOutputState) (device.SetRelayOutputStateResponse, error) {
	return device.Call_SetRelayOutputState(ctx, p.client, request)
}

// SetScopes performs the ONVIF device operation SetScopes.
//
// Profile S 7.3 Discovery, optional.
func (p *ProfileS) SetScopes(ctx context.Context, request device.SetScopes) (device.SetScopesResponse, error) {
	return device.Call_SetScopes(ctx, p.client, request)
}

// SetSynchronizationPointEvent performs the ONVIF event operation SetSynchronizationPoint.
//
// Profile S 7.7 Event Handling, optional.
func (p *ProfileS) SetSynchronizationPointEvent(ctx context.Context, request event.SetSynchronizationPoint) (event.SetSynchronizationPointResponse, error) {
	return event.Call_SetSynchronizationPoint(ctx, p.client, request)
}

// SetSynchronizationPointMedia performs the ONVIF media operation SetSynchronizationPoint.
//
// Profile S 8.1 Video Streaming – MPEG4, optional.
func (p *ProfileS) SetSynchronizationPointMedia(ctx context.Context, request media.SetSynchronizationPoint) (media.SetSynchronizationPointResponse, error) {
	return media.Call_SetSynchronizationPoint(ctx, p.client, request)
}

// SetSystemDateAndTime performs the ONVIF device operation SetSystemDateAndTime.
//
// Profile S 7.5 System, optional.
func (p *ProfileS) SetSystemDateAndTime(ctx context.Context, request device.SetSystemDateAndTime) (device.SetSystemDateAndTimeResponse, error) {
	return device.Call_SetSystemDateAndTime(ctx, p.client, request)
}

// SetSystemFactoryDefault performs the ONVIF device operation SetSystemFactoryDefault.
//
// Profile S 7.5 System, optional.
func (p *ProfileS) SetSystemFactoryDefault(ctx context.Context, request device.SetSystemFactoryDefault) (device.SetSystemFactoryDefaultResponse, error) {
	return device.Call_SetSystemFactoryDefault(ctx, p.client, request)
}

// SetUser performs the ONVIF device operation SetUser.
//
// Profile S 7.6 User Handling, mandatory for clients.
func (p *ProfileS) SetUser(ctx context.Context, request device.SetUser) (device.SetUserResponse, error) {
	return device.Call_SetUser(ctx, p.client, request)
}

// SetVideoEncoderConfiguration performs the ONVIF media operation SetVideoEncoderConfiguration.
//
// Profile S 7.10 Video Encoder Configuration, mandatory for clients.
func (p *ProfileS) SetVideoEncoderConfiguration(ctx context.Context, request media.SetVideoEncoderConfiguration) (media.SetVideoEncoderConfigurationResponse, error) {
	return media.Call_SetVideoEncoderConfiguration(ctx, p.client, request)
}

// SetVideoSourceConfiguration performs the ONVIF media operation SetVideoSourceConfiguration.
//
// Profile S 7.12 Video Source Configuration, mandatory for clients.
func (p *ProfileS) SetVideoSourceConfiguration(ctx context.Context, request media.SetVideoSourceConfiguration) (media.SetVideoSourceConfigurationResponse, error) {
	return media.Call_SetVideoSourceConfiguration(ctx, p.client, request)
}

// SetZeroConfiguration performs the ONVIF device operation SetZeroConfiguration.
//
// Profile S 8.16 Zero Configuration, mandatory for clients.
func (p *ProfileS) SetZeroConfiguration(ctx context.Context, request device.SetZeroConfiguration) (device.SetZeroConfigurationResponse, error) {
	return device.Call_SetZeroConfiguration(ctx, p.client, request)
}

// StartMulticastStreaming performs the ONVIF media operation StartMulticastStreaming.
//
// Profile S 8.12 Multicast Streaming, conditional.
func (p *ProfileS) StartMulticastStreaming(ctx context.Context, request media.StartMulticastStreaming) (media.StartMulticastStreamingResponse, error) {
	return media.Call_StartMulticastStreaming(ctx, p.client, request)
}

// Stop performs the ONVIF ptz operation Stop.
//
// Profile S 8.3 PTZ, mandatory for clients.
func (p *ProfileS) Stop(ctx context.Context, request ptz.Stop) (ptz.StopResponse, error) {
	return ptz.Call_Stop(ctx, p.client, request)
}

// StopMulticastStreaming performs the ONVIF media operation StopMulticastStreaming.
//
// Profile S 8.12 Multicast Streaming, conditional.
func (p *ProfileS) StopMulticastStreaming(ctx context.Context, request media.StopMulticastStreaming) (media.StopMulticastStreamingResponse, error) {
	return media.Call_StopMulticastStreaming(ctx, p.client, request)
}

// Subscribe performs the ONVIF event operation Subscribe.
//
// Profile S 7.7 Event Handling, mandatory for clients, subject to the footnote of that section.
func (p *ProfileS) Subscribe(ctx context.Context, request event.Subscribe) (event.SubscribeResponse, error) {
	return event.Call_Subscribe(ctx, p.client, request)
}

// SystemReboot performs the ONVIF device operation SystemReboot.
//
// Profile S 7.5 System, optional.
func (p *ProfileS) SystemReboot(ctx context.Context, request device.SystemReboot) (device.SystemRebootResponse, error) {
	return device.Call_SystemReboot(ctx, p.client, request)
}

// Unsubscribe performs the ONVIF event operation Unsubscribe.
//
// Profile S 7.7 Event Handling, optional.
func (p *ProfileS) Unsubscribe(ctx context.Context, request event.Unsubscribe) (event.UnsubscribeResponse, error) {
	return event.Call_Unsubscribe(ctx, p.client, request)
}

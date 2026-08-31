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

package sdk

import (
	"context"
	"sync"

	"github.com/jfsmig/onvif/media"
	"github.com/jfsmig/onvif/ptz"
	"github.com/jfsmig/onvif/xsd/onvif"
)

type MediaProfiles struct {
	Profiles map[onvif.ReferenceToken]*MediaProfile
}

type MediaProfile struct {
	Profile onvif.Profile
	Uris    ProfileUris
	Media   ProfileMedia
	PTZ     ProfilePTZ
}

type ProfileUris struct {
	Stream   onvif.MediaUri
	Snapshot onvif.MediaUri
}

type ProfileMedia struct {
	CompatibleMetadata []onvif.ReferenceToken

	CompatibleVideoSources   []onvif.ReferenceToken
	CompatibleVideoEncoders  []onvif.ReferenceToken
	CompatibleVideoAnalytics []onvif.ReferenceToken

	CompatibleAudioSources  []onvif.ReferenceToken
	CompatibleAudioEncoders []onvif.ReferenceToken

	CompatibleAudioOutputs  []onvif.ReferenceToken
	CompatibleAudioDecoders []onvif.ReferenceToken
}

type ProfilePTZ struct {
	Status        onvif.PTZStatus
	Configuration onvif.PTZConfiguration
	Options       onvif.PTZConfigurationOptions
	Preset        []onvif.PTZPreset
	PresetTour    []onvif.PresetTour
}

func (p *ProfileS) FetchMediaProfiles(ctx context.Context) MediaProfiles {
	out := MediaProfiles{
		Profiles: make(map[onvif.ReferenceToken]*MediaProfile),
	}

	if profiles, err := media.Call_GetProfiles(ctx, p.client, media.GetProfiles{}); err == nil {
		for _, profile := range profiles.Profiles {
			pe := p.FetchMediaProfile(ctx, profile.Token)
			out.Profiles[profile.Token] = &pe
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetProfiles").Msg("profile")
	}

	return out
}

const (
	ProtocolRTSP = onvif.TransportProtocol("RTSP")
)

const (
	StreamTypeDefault = onvif.StreamType("000")
)

func (p *ProfileS) FetchMediaProfileUris(ctx context.Context, protocol onvif.TransportProtocol, token onvif.ReferenceToken, sType onvif.StreamType) ProfileUris {
	out := ProfileUris{}

	streamRequest := media.GetStreamUri{
		StreamSetup: onvif.StreamSetup{
			Stream: sType,
			Transport: onvif.Transport{
				Protocol: protocol,
				Tunnel:   nil,
			},
		},
		ProfileToken: token,
	}

	if uris, err := media.Call_GetStreamUri(ctx, p.client, streamRequest); err == nil {
		out.Stream = uris.MediaUri
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetStreamUri").Msg("profile")
	}

	if uris, err := media.Call_GetSnapshotUri(ctx, p.client, media.GetSnapshotUri{ProfileToken: token}); err == nil {
		out.Snapshot = uris.MediaUri
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetSnapshotUri").Msg("profile")
	}

	return out
}

func (p *ProfileS) FetchMediaProfile(ctx context.Context, profileToken onvif.ReferenceToken) MediaProfile {
	out := MediaProfile{}

	if profile, err := media.Call_GetProfile(ctx, p.client, media.GetProfile{ProfileToken: profileToken}); err == nil {
		out.Profile = profile.Profile
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetProfile").Msg("profile")
	}

	out.Uris = p.FetchMediaProfileUris(ctx, ProtocolRTSP, profileToken, StreamTypeDefault)
	out.Media = p.loadProfileMedia(ctx, profileToken)
	out.PTZ = p.loadProfilePTZ(ctx, profileToken, out.Profile.PTZConfiguration.Token)

	return out
}

// loadProfilePTZ needs two different tokens. GetStatus, GetPresets and GetPresetTours are
// keyed by the profile; GetConfiguration and GetConfigurationOptions are keyed by the PTZ
// *configuration* the profile references. Passing the profile token to the latter two --
// which is what this did -- means the device is asked for a configuration that does not
// exist, so PTZ configuration never populated.
func (p *ProfileS) loadProfilePTZ(ctx context.Context, profileToken, ptzConfigToken onvif.ReferenceToken) ProfilePTZ {
	out := ProfilePTZ{}

	var wg sync.WaitGroup

	wg.Go(func() {
		if x, err := ptz.Call_GetStatus(ctx, p.client, ptz.GetStatus{ProfileToken: profileToken}); err == nil {
			out.Status = x.PTZStatus
		}
	})

	if ptzConfigToken != "" {
		wg.Go(func() {
			if x, err := ptz.Call_GetConfiguration(ctx, p.client,
				ptz.GetConfiguration{PTZConfigurationToken: ptzConfigToken}); err == nil {
				out.Configuration = x.PTZConfiguration
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetConfiguration").Msg("profile")
			}
		})

		wg.Go(func() {
			if x, err := ptz.Call_GetConfigurationOptions(ctx, p.client,
				ptz.GetConfigurationOptions{ConfigurationToken: ptzConfigToken}); err == nil {
				out.Options = x.PTZConfigurationOptions
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetConfigurationOptions").Msg("profile")
			}
		})
	}

	wg.Go(func() {
		if x, err := ptz.Call_GetPresets(ctx, p.client, ptz.GetPresets{ProfileToken: profileToken}); err == nil {
			out.Preset = x.Preset
		}
	})

	wg.Go(func() {
		if x, err := ptz.Call_GetPresetTours(ctx, p.client, ptz.GetPresetTours{ProfileToken: profileToken}); err == nil {
			out.PresetTour = x.PresetTour
		}
	})

	wg.Wait()
	return out
}

func (p *ProfileS) loadProfileMedia(ctx context.Context, profileToken onvif.ReferenceToken) ProfileMedia {
	out := ProfileMedia{}

	var wg sync.WaitGroup

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleMetadataConfigurations(ctx, p.client, media.GetCompatibleMetadataConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleMetadata = append(out.CompatibleMetadata, x.Token)
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCompatibleMetadataConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleVideoSourceConfigurations(ctx, p.client, media.GetCompatibleVideoSourceConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleVideoSources = append(out.CompatibleVideoSources, x.Token)
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoSourceConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleVideoEncoderConfigurations(ctx, p.client, media.GetCompatibleVideoEncoderConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleVideoEncoders = append(out.CompatibleVideoEncoders, x.Token)
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoEncoderConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleVideoAnalyticsConfigurations(ctx, p.client, media.GetCompatibleVideoAnalyticsConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleVideoAnalytics = append(out.CompatibleVideoAnalytics, x.Token)
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoAnalyticsConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleAudioSourceConfigurations(ctx, p.client, media.GetCompatibleAudioSourceConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleAudioSources = append(out.CompatibleAudioSources, x.Token)
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioSourceConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleAudioEncoderConfigurations(ctx, p.client, media.GetCompatibleAudioEncoderConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleAudioEncoders = append(out.CompatibleAudioEncoders, x.Token)
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioEncoderConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleAudioOutputConfigurations(ctx, p.client, media.GetCompatibleAudioOutputConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleAudioOutputs = append(out.CompatibleAudioOutputs, x.Token)
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioOutputConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {

		if all, err := media.Call_GetCompatibleAudioDecoderConfigurations(ctx, p.client, media.GetCompatibleAudioDecoderConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleAudioDecoders = append(out.CompatibleAudioDecoders, x.Token)
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioDecoderConfigurations").Msg("profile")
		}
	})

	wg.Wait()
	return out
}

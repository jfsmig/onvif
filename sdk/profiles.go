package sdk

import (
	"context"
	"github.com/jfsmig/onvif/ptz"

	"github.com/jfsmig/onvif/media"
	"github.com/jfsmig/onvif/xsd/onvif"
)

type Profiles struct {
	Profiles []XProfile
}

type XProfile struct {
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

func (dw *deviceWrapper) FetchProfiles(ctx context.Context) Profiles {
	out := Profiles{}

	if profiles, err := media.Call_GetProfiles(ctx, dw.client, media.GetProfiles{}); err == nil {
		for _, profile := range profiles.Profiles {
			pe := XProfile{Profile: profile}
			out.Profiles = append(out.Profiles, pe)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetProfiles").Msg("audio")
	}

	return out
}

const (
	ProtocolRTSP = onvif.TransportProtocol("RTSP")
)

const (
	StreamTypeDefault = onvif.StreamType("000")
)

func (dw *deviceWrapper) FetchMediaProfileUris(ctx context.Context, protocol onvif.TransportProtocol, token onvif.ReferenceToken, sType onvif.StreamType) ProfileUris {
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

	if uris, err := media.Call_GetStreamUri(ctx, dw.client, streamRequest); err == nil {
		out.Stream = uris.MediaUri
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetStreamUri").Msg("profile")
	}

	if uris, err := media.Call_GetSnapshotUri(ctx, dw.client, media.GetSnapshotUri{ProfileToken: token}); err == nil {
		out.Snapshot = uris.MediaUri
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetSnapshotUri").Msg("profile")
	}

	return out
}

func (dw *deviceWrapper) FetchProfile(ctx context.Context, profileToken onvif.ReferenceToken) XProfile {
	out := XProfile{}

	if profile, err := media.Call_GetProfile(ctx, dw.client, media.GetProfile{ProfileToken: profileToken}); err == nil {
		out.Profile = profile.Profile
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetProfile").Msg("profile")
	}

	out.Uris = dw.FetchMediaProfileUris(ctx, ProtocolRTSP, profileToken, StreamTypeDefault)
	out.Media = dw.loadProfileMedia(ctx, profileToken)
	out.PTZ = dw.loadProfilePTZ(ctx, profileToken)

	return out
}

func (dw *deviceWrapper) loadProfilePTZ(ctx context.Context, profileToken onvif.ReferenceToken) ProfilePTZ {
	out := ProfilePTZ{}

	if x, err := ptz.Call_GetStatus(ctx, dw.client, ptz.GetStatus{ProfileToken: profileToken}); err == nil {
		out.Status = x.PTZStatus
	}
	if x, err := ptz.Call_GetConfiguration(ctx, dw.client, ptz.GetConfiguration{ProfileToken: profileToken}); err == nil {
		out.Configuration = x.PTZConfiguration
	}
	if x, err := ptz.Call_GetConfigurationOptions(ctx, dw.client, ptz.GetConfigurationOptions{ProfileToken: profileToken}); err == nil {
		out.Options = x.PTZConfigurationOptions
	}
	if x, err := ptz.Call_GetPresets(ctx, dw.client, ptz.GetPresets{ProfileToken: profileToken}); err == nil {
		out.Preset = x.Preset
	}
	if x, err := ptz.Call_GetPresetTours(ctx, dw.client, ptz.GetPresetTours{ProfileToken: profileToken}); err == nil {
		out.PresetTour = x.PresetTour
	}

	return out
}

func (dw *deviceWrapper) loadProfileMedia(ctx context.Context, profileToken onvif.ReferenceToken) ProfileMedia {
	out := ProfileMedia{}

	if all, err := media.Call_GetCompatibleMetadataConfigurations(ctx, dw.client, media.GetCompatibleMetadataConfigurations{ProfileToken: profileToken}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleMetadata = append(out.CompatibleMetadata, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleMetadataConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleVideoSourceConfigurations(ctx, dw.client, media.GetCompatibleVideoSourceConfigurations{ProfileToken: profileToken}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleVideoSources = append(out.CompatibleVideoSources, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoSourceConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleVideoEncoderConfigurations(ctx, dw.client, media.GetCompatibleVideoEncoderConfigurations{ProfileToken: profileToken}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleVideoEncoders = append(out.CompatibleVideoEncoders, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoEncoderConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleVideoAnalyticsConfigurations(ctx, dw.client, media.GetCompatibleVideoAnalyticsConfigurations{ProfileToken: profileToken}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleVideoAnalytics = append(out.CompatibleVideoAnalytics, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoAnalyticsConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleAudioSourceConfigurations(ctx, dw.client, media.GetCompatibleAudioSourceConfigurations{ProfileToken: profileToken}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioSources = append(out.CompatibleAudioSources, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioSourceConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleAudioEncoderConfigurations(ctx, dw.client, media.GetCompatibleAudioEncoderConfigurations{ProfileToken: profileToken}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioEncoders = append(out.CompatibleAudioEncoders, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioEncoderConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleAudioOutputConfigurations(ctx, dw.client, media.GetCompatibleAudioOutputConfigurations{ProfileToken: profileToken}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioOutputs = append(out.CompatibleAudioOutputs, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioOutputConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleAudioDecoderConfigurations(ctx, dw.client, media.GetCompatibleAudioDecoderConfigurations{ProfileToken: profileToken}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioDecoders = append(out.CompatibleAudioDecoders, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioDecoderConfigurations").Msg("profile")
	}

	return out
}

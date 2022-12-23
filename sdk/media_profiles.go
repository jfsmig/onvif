package sdk

import (
	"context"

	"github.com/jfsmig/onvif/media"
	"github.com/jfsmig/onvif/xsd/onvif"
)

type Profiles struct {
	Profiles []XProfile
}

type XProfile struct {
	Profile onvif.Profile
	Uris    ProfileUris

	CompatibleMetadata []onvif.ReferenceToken

	CompatibleVideoSources   []onvif.ReferenceToken
	CompatibleVideoEncoders  []onvif.ReferenceToken
	CompatibleVideoAnalytics []onvif.ReferenceToken

	CompatibleAudioSources  []onvif.ReferenceToken
	CompatibleAudioEncoders []onvif.ReferenceToken

	CompatibleAudioOutputs  []onvif.ReferenceToken
	CompatibleAudioDecoders []onvif.ReferenceToken
}

type ProfileUris struct {
	Stream   onvif.MediaUri
	Snapshot onvif.MediaUri
}

func (dw *deviceWrapper) FetchMediaProfiles(ctx context.Context) Profiles {
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

func (dw *deviceWrapper) FetchMediaProfileUris(ctx context.Context, token onvif.ReferenceToken, protocol, transport string) ProfileUris {
	out := ProfileUris{}

	if uris, err := media.Call_GetStreamUri(ctx, dw.client, media.GetStreamUri{ProfileToken: token}); err == nil {
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

func (dw *deviceWrapper) FetchProfile(ctx context.Context, token onvif.ReferenceToken) XProfile {
	out := XProfile{}

	if profile, err := media.Call_GetProfile(ctx, dw.client, media.GetProfile{ProfileToken: token}); err == nil {
		out.Profile = profile.Profile
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetProfile").Msg("profile")
	}

	out.Uris = dw.FetchMediaProfileUris(ctx, token, "RTSP", "TCP")

	if all, err := media.Call_GetCompatibleMetadataConfigurations(ctx, dw.client, media.GetCompatibleMetadataConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleMetadata = append(out.CompatibleMetadata, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleMetadataConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleVideoSourceConfigurations(ctx, dw.client, media.GetCompatibleVideoSourceConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleVideoSources = append(out.CompatibleVideoSources, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoSourceConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleVideoEncoderConfigurations(ctx, dw.client, media.GetCompatibleVideoEncoderConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleVideoEncoders = append(out.CompatibleVideoEncoders, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoEncoderConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleVideoAnalyticsConfigurations(ctx, dw.client, media.GetCompatibleVideoAnalyticsConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleVideoAnalytics = append(out.CompatibleVideoAnalytics, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoAnalyticsConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleAudioSourceConfigurations(ctx, dw.client, media.GetCompatibleAudioSourceConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioSources = append(out.CompatibleAudioSources, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioSourceConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleAudioEncoderConfigurations(ctx, dw.client, media.GetCompatibleAudioEncoderConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioEncoders = append(out.CompatibleAudioEncoders, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioEncoderConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleAudioOutputConfigurations(ctx, dw.client, media.GetCompatibleAudioOutputConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioOutputs = append(out.CompatibleAudioOutputs, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioOutputConfigurations").Msg("profile")
	}

	if all, err := media.Call_GetCompatibleAudioDecoderConfigurations(ctx, dw.client, media.GetCompatibleAudioDecoderConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioDecoders = append(out.CompatibleAudioDecoders, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioDecoderConfigurations").Msg("profile")
	}

	return out
}

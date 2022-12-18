package sdk

import (
	"context"

	"github.com/use-go/onvif/media"
	"github.com/use-go/onvif/xsd/onvif"
)

type Profiles struct {
	Profiles []XProfile
}

type XProfile struct {
	Profile            onvif.Profile
	CompatibleMetadata []onvif.ReferenceToken

	CompatibleVideoSources   []onvif.ReferenceToken
	CompatibleVideoEncoders  []onvif.ReferenceToken
	CompatibleVideoAnalytics []onvif.ReferenceToken

	CompatibleAudioSources  []onvif.ReferenceToken
	CompatibleAudioEncoders []onvif.ReferenceToken

	CompatibleAudioOutputs  []onvif.ReferenceToken
	CompatibleAudioDecoders []onvif.ReferenceToken
}

func (d *deviceWrapper) FetchProfiles(ctx context.Context) Profiles {
	out := Profiles{}

	if profiles, err := media.Call_GetProfiles(ctx, d.client, media.GetProfiles{}); err == nil {
		for _, profile := range profiles.Profiles {
			pe := XProfile{Profile: profile}
			out.Profiles = append(out.Profiles, pe)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetProfiles").Msg("audio")
	}

	return out
}

func (d *deviceWrapper) FetchProfile(ctx context.Context, token onvif.ReferenceToken) XProfile {
	out := XProfile{}

	if profile, err := media.Call_GetProfile(ctx, d.client, media.GetProfile{ProfileToken: token}); err == nil {
		out.Profile = profile.Profile
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetProfile").Msg("audio")
	}

	if all, err := media.Call_GetCompatibleMetadataConfigurations(ctx, d.client, media.GetCompatibleMetadataConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleMetadata = append(out.CompatibleMetadata, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleMetadataConfigurations").Msg("audio")
	}

	if all, err := media.Call_GetCompatibleVideoSourceConfigurations(ctx, d.client, media.GetCompatibleVideoSourceConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleVideoSources = append(out.CompatibleVideoSources, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoSourceConfigurations").Msg("audio")
	}

	if all, err := media.Call_GetCompatibleVideoEncoderConfigurations(ctx, d.client, media.GetCompatibleVideoEncoderConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleVideoEncoders = append(out.CompatibleVideoEncoders, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoEncoderConfigurations").Msg("audio")
	}

	if all, err := media.Call_GetCompatibleVideoAnalyticsConfigurations(ctx, d.client, media.GetCompatibleVideoAnalyticsConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleVideoAnalytics = append(out.CompatibleVideoAnalytics, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleVideoAnalyticsConfigurations").Msg("audio")
	}

	if all, err := media.Call_GetCompatibleAudioSourceConfigurations(ctx, d.client, media.GetCompatibleAudioSourceConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioSources = append(out.CompatibleAudioSources, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioSourceConfigurations").Msg("audio")
	}

	if all, err := media.Call_GetCompatibleAudioEncoderConfigurations(ctx, d.client, media.GetCompatibleAudioEncoderConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioEncoders = append(out.CompatibleAudioEncoders, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioEncoderConfigurations").Msg("audio")
	}

	if all, err := media.Call_GetCompatibleAudioOutputConfigurations(ctx, d.client, media.GetCompatibleAudioOutputConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioOutputs = append(out.CompatibleAudioOutputs, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioOutputConfigurations").Msg("audio")
	}

	if all, err := media.Call_GetCompatibleAudioDecoderConfigurations(ctx, d.client, media.GetCompatibleAudioDecoderConfigurations{ProfileToken: token}); err == nil {
		for _, x := range all.Configurations {
			out.CompatibleAudioDecoders = append(out.CompatibleAudioDecoders, x.Token)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCompatibleAudioDecoderConfigurations").Msg("audio")
	}

	return out
}

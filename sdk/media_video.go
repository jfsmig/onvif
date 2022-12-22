package sdk

import (
	"context"

	"github.com/jfsmig/onvif/media"
	"github.com/jfsmig/onvif/xsd/onvif"
)

type Video struct {
	Sources                 []VideoSource
	Encoders                []VideoEncoderConfiguration
	AnalyticsConfigurations []onvif.VideoAnalyticsConfiguration
}

type VideoSource struct {
	Source         onvif.VideoSource
	Configurations []onvif.VideoSourceConfiguration
}

type VideoEncoderConfiguration struct {
	Configuration onvif.VideoEncoderConfiguration
	Options       onvif.VideoEncoderConfigurationOptions
}

func (d *deviceWrapper) FetchVideo(ctx context.Context) Video {
	out := Video{}

	if sources, err := media.Call_GetVideoSources(ctx, d.client, media.GetVideoSources{}); err == nil {
		for _, src := range sources.VideoSources {
			vs := VideoSource{Source: src}
			if configs, err := media.Call_GetVideoSourceConfigurations(ctx, d.client, media.GetVideoSourceConfigurations{}); err == nil {
				vs.Configurations = configs.Configurations
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetVideoSourceConfigurations").Msg("video")
			}
			out.Sources = append(out.Sources, vs)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetVideoSources").Msg("video")
	}

	if configs, err := media.Call_GetVideoAnalyticsConfigurations(ctx, d.client, media.GetVideoAnalyticsConfigurations{}); err == nil {
		for _, cfg := range configs.Configurations {
			if cfgDetail, err := media.Call_GetVideoAnalyticsConfiguration(ctx, d.client, media.GetVideoAnalyticsConfiguration{ConfigurationToken: cfg.Token}); err == nil {
				out.AnalyticsConfigurations = append(out.AnalyticsConfigurations, cfgDetail.Configuration)
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetAnalyticsConfiguration").Msg("video")
			}
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAnalyticsConfigurations").Msg("video")
	}

	if configs, err := media.Call_GetVideoEncoderConfigurations(ctx, d.client, media.GetVideoEncoderConfigurations{}); err == nil {
		for _, cfg := range configs.Configurations {
			ve := VideoEncoderConfiguration{}
			if cfgDetail, err := media.Call_GetVideoEncoderConfiguration(ctx, d.client, media.GetVideoEncoderConfiguration{ConfigurationToken: cfg.Token}); err == nil {
				ve.Configuration = cfgDetail.Configuration
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetVideoEncoderConfiguration").Msg("video")
			}
			if cfgOptions, err := media.Call_GetVideoEncoderConfigurationOptions(ctx, d.client, media.GetVideoEncoderConfigurationOptions{ConfigurationToken: cfg.Token}); err == nil {
				ve.Options = cfgOptions.Options
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetVideoEncoderConfigurationOptions").Msg("video")
			}
			out.Encoders = append(out.Encoders, ve)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetVideoEncoderConfigurations").Msg("video")
	}
	return out
}

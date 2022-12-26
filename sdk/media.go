package sdk

import (
	"context"

	"github.com/jfsmig/onvif/media"
	"github.com/jfsmig/onvif/xsd/onvif"
)

type Audio struct {
	Sources  []AudioSource
	Encoders []AudioEncoderConfiguration
	Outputs  map[onvif.ReferenceToken]*AudioOutput
}

type AudioSource struct {
	Source         onvif.AudioSource
	Configurations []onvif.AudioSourceConfiguration
}

type AudioEncoderConfiguration struct {
	Configuration onvif.AudioEncoderConfiguration
	Options       onvif.AudioEncoderConfigurationOptions
}

type AudioOutput struct {
	Output         onvif.AudioOutput
	Configurations []onvif.AudioOutputConfiguration
}

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

func (dw *deviceWrapper) FetchMedia(ctx context.Context) Media {
	out := Media{}

	if caps, err := media.Call_GetServiceCapabilities(ctx, dw.client, media.GetServiceCapabilities{}); err == nil {
		out.Capabilities = caps.Capabilities
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("media")
	}

	out.Video = dw.FetchMediaVideo(ctx)
	out.Audio = dw.FetchMediaAudio(ctx)

	return out
}

func (dw *deviceWrapper) FetchMediaVideo(ctx context.Context) Video {
	out := Video{}

	if sources, err := media.Call_GetVideoSources(ctx, dw.client, media.GetVideoSources{}); err == nil {
		for _, src := range sources.VideoSources {
			vs := VideoSource{Source: src}
			if configs, err := media.Call_GetVideoSourceConfigurations(ctx, dw.client, media.GetVideoSourceConfigurations{}); err == nil {
				vs.Configurations = configs.Configurations
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetVideoSourceConfigurations").Msg("video")
			}
			out.Sources = append(out.Sources, vs)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetVideoSources").Msg("video")
	}

	if configs, err := media.Call_GetVideoAnalyticsConfigurations(ctx, dw.client, media.GetVideoAnalyticsConfigurations{}); err == nil {
		for _, cfg := range configs.Configurations {
			if cfgDetail, err := media.Call_GetVideoAnalyticsConfiguration(ctx, dw.client, media.GetVideoAnalyticsConfiguration{ConfigurationToken: cfg.Token}); err == nil {
				out.AnalyticsConfigurations = append(out.AnalyticsConfigurations, cfgDetail.Configuration)
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetAnalyticsConfiguration").Msg("video")
			}
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAnalyticsConfigurations").Msg("video")
	}

	if configs, err := media.Call_GetVideoEncoderConfigurations(ctx, dw.client, media.GetVideoEncoderConfigurations{}); err == nil {
		for _, cfg := range configs.Configurations {
			ve := VideoEncoderConfiguration{}
			if cfgDetail, err := media.Call_GetVideoEncoderConfiguration(ctx, dw.client, media.GetVideoEncoderConfiguration{ConfigurationToken: cfg.Token}); err == nil {
				ve.Configuration = cfgDetail.Configuration
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetVideoEncoderConfiguration").Msg("video")
			}
			if cfgOptions, err := media.Call_GetVideoEncoderConfigurationOptions(ctx, dw.client, media.GetVideoEncoderConfigurationOptions{ConfigurationToken: cfg.Token}); err == nil {
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

func (dw *deviceWrapper) FetchMediaAudio(ctx context.Context) Audio {
	out := Audio{}

	if sources, err := media.Call_GetAudioSources(ctx, dw.client, media.GetAudioSources{}); err == nil {
		for _, src := range sources.AudioSources {
			vs := AudioSource{Source: src}
			if configs, err := media.Call_GetAudioSourceConfigurations(ctx, dw.client, media.GetAudioSourceConfigurations{}); err == nil {
				vs.Configurations = configs.Configurations
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetAudioSourceConfigurations").Msg("audio")
			}
			out.Sources = append(out.Sources, vs)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAudioSources").Msg("audio")
	}

	if configs, err := media.Call_GetAudioEncoderConfigurations(ctx, dw.client, media.GetAudioEncoderConfigurations{}); err == nil {
		for _, cfg := range configs.Configurations {
			ve := AudioEncoderConfiguration{}
			if cfgDetail, err := media.Call_GetAudioEncoderConfiguration(ctx, dw.client, media.GetAudioEncoderConfiguration{ConfigurationToken: cfg.Token}); err == nil {
				ve.Configuration = cfgDetail.Configuration
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetAudioEncoderConfiguration").Msg("audio")
			}
			if cfgOptions, err := media.Call_GetAudioEncoderConfigurationOptions(ctx, dw.client, media.GetAudioEncoderConfigurationOptions{ConfigurationToken: cfg.Token}); err == nil {
				ve.Options = cfgOptions.Options
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetAudioEncoderConfigurationOptions").Msg("audio")
			}
			out.Encoders = append(out.Encoders, ve)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAudioEncoderConfigurations").Msg("audio")
	}

	if outputs, err := media.Call_GetAudioOutputs(ctx, dw.client, media.GetAudioOutputs{}); err == nil {
		for _, output := range outputs.AudioOutputs {
			ao := AudioOutput{
				Output: output,
			}
			out.Outputs[output.Token] = &ao
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAnalyticsConfigurations").Msg("audio")
	}
	if configurations, err := media.Call_GetAudioOutputConfigurations(ctx, dw.client, media.GetAudioOutputConfigurations{}); err == nil {
		for _, config := range configurations.Configurations {
			out.Outputs[config.OutputToken].Configurations = append(out.Outputs[config.OutputToken].Configurations, config)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAnalyticsConfiguration").Msg("audio")
	}

	return out
}

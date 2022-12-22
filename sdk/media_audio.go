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

func (d *deviceWrapper) FetchAudio(ctx context.Context) Audio {
	out := Audio{}

	if sources, err := media.Call_GetAudioSources(ctx, d.client, media.GetAudioSources{}); err == nil {
		for _, src := range sources.AudioSources {
			vs := AudioSource{Source: src}
			if configs, err := media.Call_GetAudioSourceConfigurations(ctx, d.client, media.GetAudioSourceConfigurations{}); err == nil {
				vs.Configurations = configs.Configurations
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetAudioSourceConfigurations").Msg("audio")
			}
			out.Sources = append(out.Sources, vs)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAudioSources").Msg("audio")
	}

	if configs, err := media.Call_GetAudioEncoderConfigurations(ctx, d.client, media.GetAudioEncoderConfigurations{}); err == nil {
		for _, cfg := range configs.Configurations {
			ve := AudioEncoderConfiguration{}
			if cfgDetail, err := media.Call_GetAudioEncoderConfiguration(ctx, d.client, media.GetAudioEncoderConfiguration{ConfigurationToken: cfg.Token}); err == nil {
				ve.Configuration = cfgDetail.Configuration
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetAudioEncoderConfiguration").Msg("audio")
			}
			if cfgOptions, err := media.Call_GetAudioEncoderConfigurationOptions(ctx, d.client, media.GetAudioEncoderConfigurationOptions{ConfigurationToken: cfg.Token}); err == nil {
				ve.Options = cfgOptions.Options
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetAudioEncoderConfigurationOptions").Msg("audio")
			}
			out.Encoders = append(out.Encoders, ve)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAudioEncoderConfigurations").Msg("audio")
	}

	if outputs, err := media.Call_GetAudioOutputs(ctx, d.client, media.GetAudioOutputs{}); err == nil {
		for _, output := range outputs.AudioOutputs {
			ao := AudioOutput{
				Output: output,
			}
			out.Outputs[output.Token] = &ao
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAnalyticsConfigurations").Msg("audio")
	}
	if configurations, err := media.Call_GetAudioOutputConfigurations(ctx, d.client, media.GetAudioOutputConfigurations{}); err == nil {
		for _, config := range configurations.Configurations {
			out.Outputs[config.OutputToken].Configurations = append(out.Outputs[config.OutputToken].Configurations, config)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAnalyticsConfiguration").Msg("audio")
	}

	return out
}

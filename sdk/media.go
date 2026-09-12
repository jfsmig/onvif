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

	"github.com/jfsmig/onvif/v2/media"
	"github.com/jfsmig/onvif/v2/xsd/onvif"
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

// FetchMedia runs the service capabilities, the video tree and the audio tree
// concurrently. Three distinct fields of out, so no lock.
func (p *ProfileS) FetchMedia(ctx context.Context) Media {
	out := Media{}

	var wg sync.WaitGroup

	wg.Go(func() {
		if caps, err := media.Call_GetServiceCapabilities(ctx, p.client, media.GetServiceCapabilities{}); err == nil {
			out.Capabilities = &caps.Capabilities
		} else {
			rpcFailure(p.client, err, "GetServiceCapabilities").Msg("media")
		}
	})

	wg.Go(func() { out.Video = p.FetchMediaVideo(ctx) })
	wg.Go(func() { out.Audio = p.FetchMediaAudio(ctx) })

	wg.Wait()
	return out
}

// FetchMediaVideo runs its three independent trees concurrently: the sources, the
// analytics configurations and the encoders. Each closure owns one field of out.
func (p *ProfileS) FetchMediaVideo(ctx context.Context) Video {
	out := Video{}

	var wg sync.WaitGroup

	wg.Go(func() {
		sources, err := media.Call_GetVideoSources(ctx, p.client, media.GetVideoSources{})
		if err != nil {
			rpcFailure(p.client, err, "GetVideoSources").Msg("video")
			return
		}

		// Hoisted out of the loop below. GetVideoSourceConfigurations takes no argument and
		// returns every configuration of the service, so calling it per source made one
		// round trip per source and stored the identical list on each of them. One call now,
		// same result. The per-source list would be GetCompatibleVideoSourceConfigurations,
		// which profiles.go already uses where that is what is wanted.
		var shared []onvif.VideoSourceConfiguration
		if configs, err := media.Call_GetVideoSourceConfigurations(ctx, p.client, media.GetVideoSourceConfigurations{}); err == nil {
			shared = configs.Configurations
		} else {
			rpcFailure(p.client, err, "GetVideoSourceConfigurations").Msg("video")
		}

		for _, src := range sources.VideoSources {
			out.Sources = append(out.Sources, VideoSource{Source: src, Configurations: shared})
		}
	})

	wg.Go(func() {
		if configs, err := media.Call_GetVideoAnalyticsConfigurations(ctx, p.client, media.GetVideoAnalyticsConfigurations{}); err == nil {
			for _, cfg := range configs.Configurations {
				if cfgDetail, err := media.Call_GetVideoAnalyticsConfiguration(ctx, p.client, media.GetVideoAnalyticsConfiguration{ConfigurationToken: cfg.Token}); err == nil {
					out.AnalyticsConfigurations = append(out.AnalyticsConfigurations, cfgDetail.Configuration)
				} else {
					rpcFailure(p.client, err, "GetVideoAnalyticsConfiguration").Msg("video")
				}
			}
		} else {
			rpcFailure(p.client, err, "GetVideoAnalyticsConfigurations").Msg("video")
		}
	})

	wg.Go(func() {
		if configs, err := media.Call_GetVideoEncoderConfigurations(ctx, p.client, media.GetVideoEncoderConfigurations{}); err == nil {
			for _, cfg := range configs.Configurations {
				ve := VideoEncoderConfiguration{}
				if cfgDetail, err := media.Call_GetVideoEncoderConfiguration(ctx, p.client, media.GetVideoEncoderConfiguration{ConfigurationToken: cfg.Token}); err == nil {
					ve.Configuration = cfgDetail.Configuration
				} else {
					rpcFailure(p.client, err, "GetVideoEncoderConfiguration").Msg("video")
				}
				if cfgOptions, err := media.Call_GetVideoEncoderConfigurationOptions(ctx, p.client, media.GetVideoEncoderConfigurationOptions{ConfigurationToken: cfg.Token}); err == nil {
					ve.Options = cfgOptions.Options
				} else {
					rpcFailure(p.client, err, "GetVideoEncoderConfigurationOptions").Msg("video")
				}
				out.Encoders = append(out.Encoders, ve)
			}
		} else {
			rpcFailure(p.client, err, "GetVideoEncoderConfigurations").Msg("video")
		}
	})

	wg.Wait()
	return out
}

func (p *ProfileS) FetchMediaAudio(ctx context.Context) Audio {
	out := Audio{
		Outputs: make(map[onvif.ReferenceToken]*AudioOutput),
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		sources, err := media.Call_GetAudioSources(ctx, p.client, media.GetAudioSources{})
		if err != nil {
			rpcFailure(p.client, err, "GetAudioSources").Msg("audio")
			return
		}

		// Hoisted, as in FetchMediaVideo: GetAudioSourceConfigurations takes no argument and
		// returns every configuration of the service, so calling it per source cost one
		// round trip each to store the same list.
		var shared []onvif.AudioSourceConfiguration
		if configs, err := media.Call_GetAudioSourceConfigurations(ctx, p.client, media.GetAudioSourceConfigurations{}); err == nil {
			shared = configs.Configurations
		} else {
			rpcFailure(p.client, err, "GetAudioSourceConfigurations").Msg("audio")
		}

		for _, src := range sources.AudioSources {
			out.Sources = append(out.Sources, AudioSource{Source: src, Configurations: shared})
		}
	})

	wg.Go(func() {
		if configs, err := media.Call_GetAudioEncoderConfigurations(ctx, p.client, media.GetAudioEncoderConfigurations{}); err == nil {
			for _, cfg := range configs.Configurations {
				ve := AudioEncoderConfiguration{}
				if cfgDetail, err := media.Call_GetAudioEncoderConfiguration(ctx, p.client, media.GetAudioEncoderConfiguration{ConfigurationToken: cfg.Token}); err == nil {
					ve.Configuration = cfgDetail.Configuration
				} else {
					rpcFailure(p.client, err, "GetAudioEncoderConfiguration").Msg("audio")
				}
				if cfgOptions, err := media.Call_GetAudioEncoderConfigurationOptions(ctx, p.client, media.GetAudioEncoderConfigurationOptions{ConfigurationToken: cfg.Token}); err == nil {
					ve.Options = cfgOptions.Options
				} else {
					rpcFailure(p.client, err, "GetAudioEncoderConfigurationOptions").Msg("audio")
				}
				out.Encoders = append(out.Encoders, ve)
			}
		} else {
			rpcFailure(p.client, err, "GetAudioEncoderConfigurations").Msg("audio")
		}
	})

	// These two stay in one closure, in this order, and that is not an oversight.
	// GetAudioOutputs fills out.Outputs and GetAudioOutputConfigurations then reads back
	// what it wrote, to attach each configuration to its output. Splitting them across
	// goroutines would be a map race and would also lose the association, undoing the fix
	// in 45bbd6a. They are one unit of work, so they get one goroutine.
	wg.Go(func() {
		if outputs, err := media.Call_GetAudioOutputs(ctx, p.client, media.GetAudioOutputs{}); err == nil {
			for _, output := range outputs.AudioOutputs {
				ao := AudioOutput{
					Output: output,
				}
				out.Outputs[output.Token] = &ao
			}
		} else {
			rpcFailure(p.client, err, "GetAudioOutputs").Msg("audio")
		}

		if configurations, err := media.Call_GetAudioOutputConfigurations(ctx, p.client, media.GetAudioOutputConfigurations{}); err == nil {
			for _, config := range configurations.Configurations {
				ao, found := out.Outputs[config.OutputToken]
				if !found {
					ao = &AudioOutput{}
					out.Outputs[config.OutputToken] = ao
				}
				ao.Configurations = append(ao.Configurations, config)
			}
		} else {
			rpcFailure(p.client, err, "GetAudioOutputConfigurations").Msg("audio")
		}
	})

	wg.Wait()
	return out
}

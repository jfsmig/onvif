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
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/ptz"
	"github.com/jfsmig/onvif/xsd"
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

// FetchMediaProfiles hydrates every media profile of the appliance, all of them at once.
//
// Each profile costs about thirteen round trips, so doing them in sequence made the total
// grow with the number of profiles: a four-profile camera spent some fifty exchanges inside
// the one-minute budget bin/onvif-cli runs under, and the deadline expired before the last
// profile was reached.
//
// The map is filled with empty entries first and is then only read, so each goroutine
// writes through its own pointer and no lock is needed -- the same one-writer-per-
// destination rule the other fan-outs here rest on.
func (p *ProfileS) FetchMediaProfiles(ctx context.Context) MediaProfiles {
	out := MediaProfiles{
		Profiles: make(map[onvif.ReferenceToken]*MediaProfile),
	}

	profiles, err := media.Call_GetProfiles(ctx, p.client, media.GetProfiles{})
	if err != nil {
		rpcFailure(p.client, err, "GetProfiles").Msg("profile")
		return out
	}

	for _, profile := range profiles.Profiles {
		// A malformed reply repeating a token must not hand the same entry to two
		// goroutines.
		if _, duplicate := out.Profiles[profile.Token]; duplicate {
			Logger.Trace().Str("token", string(profile.Token)).Msg("duplicate media profile token")
			continue
		}
		out.Profiles[profile.Token] = &MediaProfile{}
	}

	var wg sync.WaitGroup
	for token, entry := range out.Profiles {
		wg.Go(func() { *entry = p.FetchMediaProfile(ctx, token) })
	}
	wg.Wait()

	return out
}

const (
	ProtocolRTSP = onvif.TransportProtocol("RTSP")
)

// StreamTypeDefault is the StreamSetup.Stream value sent with every GetStreamUri here.
//
// The value is not a spelling docs/wsdl/media.wsdl uses: its three documented setups all give
// StreamType as "RTP_unicast", and "000" appears nowhere in that file. It is inherited from
// upstream and it works against the cameras this has been run on, which is the only evidence
// there is -- tt:StreamType is declared in onvif.xsd, and that schema is not vendored here
// (see docs/README.md), so nothing in the tree can say whether "000" is in its enumeration.
//
// Left as it stands rather than corrected blind: changing what every stream request sends, on
// the strength of a document we do not have, risks breaking the cameras it currently works
// with. Confirm against real hardware before touching it.
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
		// Redacted, because these two URIs are the whole output of `onvif-cli streams` and
		// are JSON-encoded by `dump profile`: firmware commonly answers with the account
		// spliced in, as rtsp://admin:secret@host/..., and AGENTS.md's rule is that a secret
		// must not reach a log or a dump. It is the same rule networking.AddEndpoint and
		// AtDeviceHost apply to every other URI a device hands over; this family was the one
		// it had never reached, and the one printed by default.
		//
		// The cost is stated where a caller will meet it, on FetchStreamURI: a URI from here
		// will not authenticate on its own, and a caller that needs authenticated RTSP adds
		// its own credentials at the point of use, where it can decide how they are handled.
		out.Stream = uris.MediaUri
		out.Stream.Uri = xsd.AnyURI(networking.WithoutUserinfo(string(uris.MediaUri.Uri)))
	} else {
		rpcFailure(p.client, err, "GetStreamUri").Msg("profile")
	}

	if uris, err := media.Call_GetSnapshotUri(ctx, p.client, media.GetSnapshotUri{ProfileToken: token}); err == nil {
		out.Snapshot = uris.MediaUri
		out.Snapshot.Uri = xsd.AnyURI(networking.WithoutUserinfo(string(uris.MediaUri.Uri)))
	} else {
		rpcFailure(p.client, err, "GetSnapshotUri").Msg("profile")
	}

	return out
}

func (p *ProfileS) FetchMediaProfile(ctx context.Context, profileToken onvif.ReferenceToken) MediaProfile {
	out := MediaProfile{}

	if profile, err := media.Call_GetProfile(ctx, p.client, media.GetProfile{ProfileToken: profileToken}); err == nil {
		out.Profile = profile.Profile
	} else {
		rpcFailure(p.client, err, "GetProfile").Msg("profile")
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
		} else {
			rpcFailure(p.client, err, "GetStatus").Msg("profile")
		}
	})

	if ptzConfigToken != "" {
		wg.Go(func() {
			if x, err := ptz.Call_GetConfiguration(ctx, p.client,
				ptz.GetConfiguration{PTZConfigurationToken: ptzConfigToken}); err == nil {
				out.Configuration = x.PTZConfiguration
			} else {
				rpcFailure(p.client, err, "GetConfiguration").Msg("profile")
			}
		})

		wg.Go(func() {
			if x, err := ptz.Call_GetConfigurationOptions(ctx, p.client,
				ptz.GetConfigurationOptions{ConfigurationToken: ptzConfigToken}); err == nil {
				out.Options = x.PTZConfigurationOptions
			} else {
				rpcFailure(p.client, err, "GetConfigurationOptions").Msg("profile")
			}
		})
	}

	wg.Go(func() {
		if x, err := ptz.Call_GetPresets(ctx, p.client, ptz.GetPresets{ProfileToken: profileToken}); err == nil {
			out.Preset = x.Preset
		} else {
			rpcFailure(p.client, err, "GetPresets").Msg("profile")
		}
	})

	wg.Go(func() {
		if x, err := ptz.Call_GetPresetTours(ctx, p.client, ptz.GetPresetTours{ProfileToken: profileToken}); err == nil {
			out.PresetTour = x.PresetTour
		} else {
			rpcFailure(p.client, err, "GetPresetTours").Msg("profile")
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
			rpcFailure(p.client, err, "GetCompatibleMetadataConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleVideoSourceConfigurations(ctx, p.client, media.GetCompatibleVideoSourceConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleVideoSources = append(out.CompatibleVideoSources, x.Token)
			}
		} else {
			rpcFailure(p.client, err, "GetCompatibleVideoSourceConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleVideoEncoderConfigurations(ctx, p.client, media.GetCompatibleVideoEncoderConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleVideoEncoders = append(out.CompatibleVideoEncoders, x.Token)
			}
		} else {
			rpcFailure(p.client, err, "GetCompatibleVideoEncoderConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleVideoAnalyticsConfigurations(ctx, p.client, media.GetCompatibleVideoAnalyticsConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleVideoAnalytics = append(out.CompatibleVideoAnalytics, x.Token)
			}
		} else {
			rpcFailure(p.client, err, "GetCompatibleVideoAnalyticsConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleAudioSourceConfigurations(ctx, p.client, media.GetCompatibleAudioSourceConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleAudioSources = append(out.CompatibleAudioSources, x.Token)
			}
		} else {
			rpcFailure(p.client, err, "GetCompatibleAudioSourceConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleAudioEncoderConfigurations(ctx, p.client, media.GetCompatibleAudioEncoderConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleAudioEncoders = append(out.CompatibleAudioEncoders, x.Token)
			}
		} else {
			rpcFailure(p.client, err, "GetCompatibleAudioEncoderConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {
		if all, err := media.Call_GetCompatibleAudioOutputConfigurations(ctx, p.client, media.GetCompatibleAudioOutputConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleAudioOutputs = append(out.CompatibleAudioOutputs, x.Token)
			}
		} else {
			rpcFailure(p.client, err, "GetCompatibleAudioOutputConfigurations").Msg("profile")
		}
	})

	wg.Go(func() {

		if all, err := media.Call_GetCompatibleAudioDecoderConfigurations(ctx, p.client, media.GetCompatibleAudioDecoderConfigurations{ProfileToken: profileToken}); err == nil {
			for _, x := range all.Configurations {
				out.CompatibleAudioDecoders = append(out.CompatibleAudioDecoders, x.Token)
			}
		} else {
			rpcFailure(p.client, err, "GetCompatibleAudioDecoderConfigurations").Msg("profile")
		}
	})

	wg.Wait()
	return out
}

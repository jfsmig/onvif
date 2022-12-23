// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAudioSources forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioSourcesResponse.
func Call_GetAudioSources(ctx context.Context, dev *networking.Client, request GetAudioSources) (GetAudioSourcesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioSourcesResponse GetAudioSourcesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetAudioSourcesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioSources")
		return reply.Body.GetAudioSourcesResponse, err
	}
}

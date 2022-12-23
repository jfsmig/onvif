// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAudioSourceConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioSourceConfigurationResponse.
func Call_GetAudioSourceConfiguration(ctx context.Context, dev *networking.Client, request GetAudioSourceConfiguration) (GetAudioSourceConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioSourceConfigurationResponse GetAudioSourceConfigurationResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetAudioSourceConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioSourceConfiguration")
		return reply.Body.GetAudioSourceConfigurationResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAudioSourceConfigurationOptions forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioSourceConfigurationOptionsResponse.
func Call_GetAudioSourceConfigurationOptions(ctx context.Context, dev *networking.Client, request GetAudioSourceConfigurationOptions) (GetAudioSourceConfigurationOptionsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioSourceConfigurationOptionsResponse GetAudioSourceConfigurationOptionsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetAudioSourceConfigurationOptionsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioSourceConfigurationOptions")
		return reply.Body.GetAudioSourceConfigurationOptionsResponse, err
	}
}

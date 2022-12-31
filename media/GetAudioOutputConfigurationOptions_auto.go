// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAudioOutputConfigurationOptions forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioOutputConfigurationOptionsResponse.
func Call_GetAudioOutputConfigurationOptions(ctx context.Context, dev *networking.Client, request GetAudioOutputConfigurationOptions) (GetAudioOutputConfigurationOptionsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioOutputConfigurationOptionsResponse GetAudioOutputConfigurationOptionsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetAudioOutputConfigurationOptionsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioOutputConfigurationOptions")
		return reply.Body.GetAudioOutputConfigurationOptionsResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAudioOutputConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioOutputConfigurationsResponse.
func Call_GetAudioOutputConfigurations(ctx context.Context, dev *networking.Client, request GetAudioOutputConfigurations) (GetAudioOutputConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioOutputConfigurationsResponse GetAudioOutputConfigurationsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetAudioOutputConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioOutputConfigurations")
		return reply.Body.GetAudioOutputConfigurationsResponse, err
	}
}

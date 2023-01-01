// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetCompatibleAudioOutputConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCompatibleAudioOutputConfigurationsResponse.
func Call_GetCompatibleAudioOutputConfigurations(ctx context.Context, dev *networking.Client, request GetCompatibleAudioOutputConfigurations) (GetCompatibleAudioOutputConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCompatibleAudioOutputConfigurationsResponse GetCompatibleAudioOutputConfigurationsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetCompatibleAudioOutputConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCompatibleAudioOutputConfigurations")
		return reply.Body.GetCompatibleAudioOutputConfigurationsResponse, err
	}
}

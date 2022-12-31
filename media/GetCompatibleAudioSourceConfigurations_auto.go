// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetCompatibleAudioSourceConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCompatibleAudioSourceConfigurationsResponse.
func Call_GetCompatibleAudioSourceConfigurations(ctx context.Context, dev *networking.Client, request GetCompatibleAudioSourceConfigurations) (GetCompatibleAudioSourceConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCompatibleAudioSourceConfigurationsResponse GetCompatibleAudioSourceConfigurationsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetCompatibleAudioSourceConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCompatibleAudioSourceConfigurations")
		return reply.Body.GetCompatibleAudioSourceConfigurationsResponse, err
	}
}

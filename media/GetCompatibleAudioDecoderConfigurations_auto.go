// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetCompatibleAudioDecoderConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCompatibleAudioDecoderConfigurationsResponse.
func Call_GetCompatibleAudioDecoderConfigurations(ctx context.Context, dev *networking.Client, request GetCompatibleAudioDecoderConfigurations) (GetCompatibleAudioDecoderConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCompatibleAudioDecoderConfigurationsResponse GetCompatibleAudioDecoderConfigurationsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetCompatibleAudioDecoderConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCompatibleAudioDecoderConfigurations")
		return reply.Body.GetCompatibleAudioDecoderConfigurationsResponse, err
	}
}

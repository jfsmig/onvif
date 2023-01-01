// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetCompatibleAudioEncoderConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCompatibleAudioEncoderConfigurationsResponse.
func Call_GetCompatibleAudioEncoderConfigurations(ctx context.Context, dev *networking.Client, request GetCompatibleAudioEncoderConfigurations) (GetCompatibleAudioEncoderConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCompatibleAudioEncoderConfigurationsResponse GetCompatibleAudioEncoderConfigurationsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetCompatibleAudioEncoderConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCompatibleAudioEncoderConfigurations")
		return reply.Body.GetCompatibleAudioEncoderConfigurationsResponse, err
	}
}

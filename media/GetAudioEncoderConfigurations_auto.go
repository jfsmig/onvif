// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAudioEncoderConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioEncoderConfigurationsResponse.
func Call_GetAudioEncoderConfigurations(ctx context.Context, dev *networking.Client, request GetAudioEncoderConfigurations) (GetAudioEncoderConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioEncoderConfigurationsResponse GetAudioEncoderConfigurationsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetAudioEncoderConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioEncoderConfigurations")
		return reply.Body.GetAudioEncoderConfigurationsResponse, err
	}
}

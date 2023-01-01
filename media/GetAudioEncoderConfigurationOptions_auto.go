// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAudioEncoderConfigurationOptions forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioEncoderConfigurationOptionsResponse.
func Call_GetAudioEncoderConfigurationOptions(ctx context.Context, dev *networking.Client, request GetAudioEncoderConfigurationOptions) (GetAudioEncoderConfigurationOptionsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioEncoderConfigurationOptionsResponse GetAudioEncoderConfigurationOptionsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetAudioEncoderConfigurationOptionsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioEncoderConfigurationOptions")
		return reply.Body.GetAudioEncoderConfigurationOptionsResponse, err
	}
}

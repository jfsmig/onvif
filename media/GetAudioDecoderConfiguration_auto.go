// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAudioDecoderConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioDecoderConfigurationResponse.
func Call_GetAudioDecoderConfiguration(ctx context.Context, dev *networking.Client, request GetAudioDecoderConfiguration) (GetAudioDecoderConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioDecoderConfigurationResponse GetAudioDecoderConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetAudioDecoderConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioDecoderConfiguration")
		return reply.Body.GetAudioDecoderConfigurationResponse, err
	}
}

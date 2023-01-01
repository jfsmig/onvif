// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetAudioEncoderConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a SetAudioEncoderConfigurationResponse.
func Call_SetAudioEncoderConfiguration(ctx context.Context, dev *networking.Client, request SetAudioEncoderConfiguration) (SetAudioEncoderConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetAudioEncoderConfigurationResponse SetAudioEncoderConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.SetAudioEncoderConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetAudioEncoderConfiguration")
		return reply.Body.SetAudioEncoderConfigurationResponse, err
	}
}

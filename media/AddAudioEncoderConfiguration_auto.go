// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_AddAudioEncoderConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a AddAudioEncoderConfigurationResponse.
func Call_AddAudioEncoderConfiguration(ctx context.Context, dev *networking.Client, request AddAudioEncoderConfiguration) (AddAudioEncoderConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			AddAudioEncoderConfigurationResponse AddAudioEncoderConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.AddAudioEncoderConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "AddAudioEncoderConfiguration")
		return reply.Body.AddAudioEncoderConfigurationResponse, err
	}
}

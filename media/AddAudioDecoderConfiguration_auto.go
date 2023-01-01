// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_AddAudioDecoderConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a AddAudioDecoderConfigurationResponse.
func Call_AddAudioDecoderConfiguration(ctx context.Context, dev *networking.Client, request AddAudioDecoderConfiguration) (AddAudioDecoderConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			AddAudioDecoderConfigurationResponse AddAudioDecoderConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.AddAudioDecoderConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "AddAudioDecoderConfiguration")
		return reply.Body.AddAudioDecoderConfigurationResponse, err
	}
}

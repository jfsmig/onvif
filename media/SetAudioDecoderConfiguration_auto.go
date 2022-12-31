// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetAudioDecoderConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a SetAudioDecoderConfigurationResponse.
func Call_SetAudioDecoderConfiguration(ctx context.Context, dev *networking.Client, request SetAudioDecoderConfiguration) (SetAudioDecoderConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetAudioDecoderConfigurationResponse SetAudioDecoderConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SetAudioDecoderConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetAudioDecoderConfiguration")
		return reply.Body.SetAudioDecoderConfigurationResponse, err
	}
}

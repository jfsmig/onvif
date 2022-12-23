// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_RemoveAudioDecoderConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a RemoveAudioDecoderConfigurationResponse.
func Call_RemoveAudioDecoderConfiguration(ctx context.Context, dev *networking.Client, request RemoveAudioDecoderConfiguration) (RemoveAudioDecoderConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			RemoveAudioDecoderConfigurationResponse RemoveAudioDecoderConfigurationResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.RemoveAudioDecoderConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "RemoveAudioDecoderConfiguration")
		return reply.Body.RemoveAudioDecoderConfigurationResponse, err
	}
}

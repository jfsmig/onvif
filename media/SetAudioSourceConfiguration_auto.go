// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetAudioSourceConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a SetAudioSourceConfigurationResponse.
func Call_SetAudioSourceConfiguration(ctx context.Context, dev *networking.Client, request SetAudioSourceConfiguration) (SetAudioSourceConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetAudioSourceConfigurationResponse SetAudioSourceConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SetAudioSourceConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetAudioSourceConfiguration")
		return reply.Body.SetAudioSourceConfigurationResponse, err
	}
}

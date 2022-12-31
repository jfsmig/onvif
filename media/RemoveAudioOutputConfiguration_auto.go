// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_RemoveAudioOutputConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a RemoveAudioOutputConfigurationResponse.
func Call_RemoveAudioOutputConfiguration(ctx context.Context, dev *networking.Client, request RemoveAudioOutputConfiguration) (RemoveAudioOutputConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			RemoveAudioOutputConfigurationResponse RemoveAudioOutputConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.RemoveAudioOutputConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "RemoveAudioOutputConfiguration")
		return reply.Body.RemoveAudioOutputConfigurationResponse, err
	}
}

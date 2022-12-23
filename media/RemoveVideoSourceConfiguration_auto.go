// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_RemoveVideoSourceConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a RemoveVideoSourceConfigurationResponse.
func Call_RemoveVideoSourceConfiguration(ctx context.Context, dev *networking.Client, request RemoveVideoSourceConfiguration) (RemoveVideoSourceConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			RemoveVideoSourceConfigurationResponse RemoveVideoSourceConfigurationResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.RemoveVideoSourceConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "RemoveVideoSourceConfiguration")
		return reply.Body.RemoveVideoSourceConfigurationResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetVideoEncoderConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a SetVideoEncoderConfigurationResponse.
func Call_SetVideoEncoderConfiguration(ctx context.Context, dev *networking.Client, request SetVideoEncoderConfiguration) (SetVideoEncoderConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetVideoEncoderConfigurationResponse SetVideoEncoderConfigurationResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.SetVideoEncoderConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetVideoEncoderConfiguration")
		return reply.Body.SetVideoEncoderConfigurationResponse, err
	}
}

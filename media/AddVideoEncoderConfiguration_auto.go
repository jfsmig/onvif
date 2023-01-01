// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_AddVideoEncoderConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a AddVideoEncoderConfigurationResponse.
func Call_AddVideoEncoderConfiguration(ctx context.Context, dev *networking.Client, request AddVideoEncoderConfiguration) (AddVideoEncoderConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			AddVideoEncoderConfigurationResponse AddVideoEncoderConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.AddVideoEncoderConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "AddVideoEncoderConfiguration")
		return reply.Body.AddVideoEncoderConfigurationResponse, err
	}
}

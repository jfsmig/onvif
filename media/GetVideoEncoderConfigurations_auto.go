// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetVideoEncoderConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetVideoEncoderConfigurationsResponse.
func Call_GetVideoEncoderConfigurations(ctx context.Context, dev *networking.Client, request GetVideoEncoderConfigurations) (GetVideoEncoderConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetVideoEncoderConfigurationsResponse GetVideoEncoderConfigurationsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetVideoEncoderConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetVideoEncoderConfigurations")
		return reply.Body.GetVideoEncoderConfigurationsResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetVideoEncoderConfigurationOptions forwards the call to dev.CallMethod() then parses the payload of the reply as a GetVideoEncoderConfigurationOptionsResponse.
func Call_GetVideoEncoderConfigurationOptions(ctx context.Context, dev *networking.Client, request GetVideoEncoderConfigurationOptions) (GetVideoEncoderConfigurationOptionsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetVideoEncoderConfigurationOptionsResponse GetVideoEncoderConfigurationOptionsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetVideoEncoderConfigurationOptionsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetVideoEncoderConfigurationOptions")
		return reply.Body.GetVideoEncoderConfigurationOptionsResponse, err
	}
}

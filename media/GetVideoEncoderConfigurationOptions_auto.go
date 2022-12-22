// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/juju/errors"
	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetVideoEncoderConfigurationOptions forwards the call to dev.CallMethod() then parses the payload of the reply as a GetVideoEncoderConfigurationOptionsResponse.
func Call_GetVideoEncoderConfigurationOptions(ctx context.Context, dev *device.Device, request GetVideoEncoderConfigurationOptions) (GetVideoEncoderConfigurationOptionsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetVideoEncoderConfigurationOptionsResponse GetVideoEncoderConfigurationOptionsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetVideoEncoderConfigurationOptionsResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetVideoEncoderConfigurationOptions")
		return reply.Body.GetVideoEncoderConfigurationOptionsResponse, errors.Annotate(err, "reply")
	}
}

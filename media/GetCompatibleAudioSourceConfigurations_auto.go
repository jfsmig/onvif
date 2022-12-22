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

// Call_GetCompatibleAudioSourceConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCompatibleAudioSourceConfigurationsResponse.
func Call_GetCompatibleAudioSourceConfigurations(ctx context.Context, dev *device.Device, request GetCompatibleAudioSourceConfigurations) (GetCompatibleAudioSourceConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCompatibleAudioSourceConfigurationsResponse GetCompatibleAudioSourceConfigurationsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetCompatibleAudioSourceConfigurationsResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCompatibleAudioSourceConfigurations")
		return reply.Body.GetCompatibleAudioSourceConfigurationsResponse, errors.Annotate(err, "reply")
	}
}
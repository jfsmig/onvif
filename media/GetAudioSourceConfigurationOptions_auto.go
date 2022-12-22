// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/juju/errors"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAudioSourceConfigurationOptions forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioSourceConfigurationOptionsResponse.
func Call_GetAudioSourceConfigurationOptions(ctx context.Context, dev *networking.Client, request GetAudioSourceConfigurationOptions) (GetAudioSourceConfigurationOptionsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioSourceConfigurationOptionsResponse GetAudioSourceConfigurationOptionsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetAudioSourceConfigurationOptionsResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioSourceConfigurationOptions")
		return reply.Body.GetAudioSourceConfigurationOptionsResponse, errors.Annotate(err, "reply")
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/juju/errors"
	"github.com/use-go/onvif/networking"
)

// Call_GetAudioOutputConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAudioOutputConfigurationResponse.
func Call_GetAudioOutputConfiguration(ctx context.Context, dev *networking.Client, request GetAudioOutputConfiguration) (GetAudioOutputConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAudioOutputConfigurationResponse GetAudioOutputConfigurationResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetAudioOutputConfigurationResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAudioOutputConfiguration")
		return reply.Body.GetAudioOutputConfigurationResponse, errors.Annotate(err, "reply")
	}
}

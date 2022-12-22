// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/juju/errors"
	"github.com/use-go/onvif/networking"
)

// Call_AddAudioSourceConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a AddAudioSourceConfigurationResponse.
func Call_AddAudioSourceConfiguration(ctx context.Context, dev *networking.Client, request AddAudioSourceConfiguration) (AddAudioSourceConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			AddAudioSourceConfigurationResponse AddAudioSourceConfigurationResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.AddAudioSourceConfigurationResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "AddAudioSourceConfiguration")
		return reply.Body.AddAudioSourceConfigurationResponse, errors.Annotate(err, "reply")
	}
}

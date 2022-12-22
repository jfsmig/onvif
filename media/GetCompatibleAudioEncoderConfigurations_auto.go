// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/juju/errors"
	"github.com/use-go/onvif/networking"
)

// Call_GetCompatibleAudioEncoderConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCompatibleAudioEncoderConfigurationsResponse.
func Call_GetCompatibleAudioEncoderConfigurations(ctx context.Context, dev *networking.Client, request GetCompatibleAudioEncoderConfigurations) (GetCompatibleAudioEncoderConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCompatibleAudioEncoderConfigurationsResponse GetCompatibleAudioEncoderConfigurationsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetCompatibleAudioEncoderConfigurationsResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCompatibleAudioEncoderConfigurations")
		return reply.Body.GetCompatibleAudioEncoderConfigurationsResponse, errors.Annotate(err, "reply")
	}
}

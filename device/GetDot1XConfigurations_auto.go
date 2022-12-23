// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetDot1XConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetDot1XConfigurationsResponse.
func Call_GetDot1XConfigurations(ctx context.Context, dev *networking.Client, request GetDot1XConfigurations) (GetDot1XConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetDot1XConfigurationsResponse GetDot1XConfigurationsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetDot1XConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetDot1XConfigurations")
		return reply.Body.GetDot1XConfigurationsResponse, err
	}
}

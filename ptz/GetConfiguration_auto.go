// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package ptz

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a GetConfigurationResponse.
func Call_GetConfiguration(ctx context.Context, dev *networking.Client, request GetConfiguration) (GetConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetConfigurationResponse GetConfigurationResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetConfiguration")
		return reply.Body.GetConfigurationResponse, err
	}
}

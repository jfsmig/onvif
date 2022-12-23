// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetCapabilities forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCapabilitiesResponse.
func Call_GetCapabilities(ctx context.Context, dev *networking.Client, request GetCapabilities) (GetCapabilitiesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCapabilitiesResponse GetCapabilitiesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetCapabilitiesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCapabilities")
		return reply.Body.GetCapabilitiesResponse, err
	}
}

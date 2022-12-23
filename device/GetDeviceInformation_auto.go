// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetDeviceInformation forwards the call to dev.CallMethod() then parses the payload of the reply as a GetDeviceInformationResponse.
func Call_GetDeviceInformation(ctx context.Context, dev *networking.Client, request GetDeviceInformation) (GetDeviceInformationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetDeviceInformationResponse GetDeviceInformationResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetDeviceInformationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetDeviceInformation")
		return reply.Body.GetDeviceInformationResponse, err
	}
}

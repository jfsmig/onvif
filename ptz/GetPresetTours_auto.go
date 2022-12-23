// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package ptz

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetPresetTours forwards the call to dev.CallMethod() then parses the payload of the reply as a GetPresetToursResponse.
func Call_GetPresetTours(ctx context.Context, dev *networking.Client, request GetPresetTours) (GetPresetToursResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetPresetToursResponse GetPresetToursResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetPresetToursResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetPresetTours")
		return reply.Body.GetPresetToursResponse, err
	}
}

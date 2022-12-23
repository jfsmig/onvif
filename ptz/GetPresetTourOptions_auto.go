// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package ptz

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetPresetTourOptions forwards the call to dev.CallMethod() then parses the payload of the reply as a GetPresetTourOptionsResponse.
func Call_GetPresetTourOptions(ctx context.Context, dev *networking.Client, request GetPresetTourOptions) (GetPresetTourOptionsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetPresetTourOptionsResponse GetPresetTourOptionsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetPresetTourOptionsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetPresetTourOptions")
		return reply.Body.GetPresetTourOptionsResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetVideoSourceModes forwards the call to dev.CallMethod() then parses the payload of the reply as a GetVideoSourceModesResponse.
func Call_GetVideoSourceModes(ctx context.Context, dev *networking.Client, request GetVideoSourceModes) (GetVideoSourceModesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetVideoSourceModesResponse GetVideoSourceModesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetVideoSourceModesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetVideoSourceModes")
		return reply.Body.GetVideoSourceModesResponse, err
	}
}

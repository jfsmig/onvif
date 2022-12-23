// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_CreateOSD forwards the call to dev.CallMethod() then parses the payload of the reply as a CreateOSDResponse.
func Call_CreateOSD(ctx context.Context, dev *networking.Client, request CreateOSD) (CreateOSDResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			CreateOSDResponse CreateOSDResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.CreateOSDResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "CreateOSD")
		return reply.Body.CreateOSDResponse, err
	}
}

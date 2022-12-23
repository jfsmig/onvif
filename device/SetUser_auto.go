// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetUser forwards the call to dev.CallMethod() then parses the payload of the reply as a SetUserResponse.
func Call_SetUser(ctx context.Context, dev *networking.Client, request SetUser) (SetUserResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetUserResponse SetUserResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.SetUserResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetUser")
		return reply.Body.SetUserResponse, err
	}
}

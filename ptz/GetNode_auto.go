// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package ptz

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetNode forwards the call to dev.CallMethod() then parses the payload of the reply as a GetNodeResponse.
func Call_GetNode(ctx context.Context, dev *networking.Client, request GetNode) (GetNodeResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetNodeResponse GetNodeResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetNodeResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetNode")
		return reply.Body.GetNodeResponse, err
	}
}

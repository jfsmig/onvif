// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package event

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetSynchronizationPoint forwards the call to dev.CallMethod() then parses the payload of the reply as a SetSynchronizationPointResponse.
func Call_SetSynchronizationPoint(ctx context.Context, dev *networking.Client, request SetSynchronizationPoint) (SetSynchronizationPointResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetSynchronizationPointResponse SetSynchronizationPointResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SetSynchronizationPointResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetSynchronizationPoint")
		return reply.Body.SetSynchronizationPointResponse, err
	}
}

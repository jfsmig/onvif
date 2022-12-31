// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetVideoSourceMode forwards the call to dev.CallMethod() then parses the payload of the reply as a SetVideoSourceModeResponse.
func Call_SetVideoSourceMode(ctx context.Context, dev *networking.Client, request SetVideoSourceMode) (SetVideoSourceModeResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetVideoSourceModeResponse SetVideoSourceModeResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SetVideoSourceModeResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetVideoSourceMode")
		return reply.Body.SetVideoSourceModeResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetRelayOutputState forwards the call to dev.CallMethod() then parses the payload of the reply as a SetRelayOutputStateResponse.
func Call_SetRelayOutputState(ctx context.Context, dev *networking.Client, request SetRelayOutputState) (SetRelayOutputStateResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetRelayOutputStateResponse SetRelayOutputStateResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.SetRelayOutputStateResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetRelayOutputState")
		return reply.Body.SetRelayOutputStateResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetHostname forwards the call to dev.CallMethod() then parses the payload of the reply as a SetHostnameResponse.
func Call_SetHostname(ctx context.Context, dev *networking.Client, request SetHostname) (SetHostnameResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetHostnameResponse SetHostnameResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.SetHostnameResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetHostname")
		return reply.Body.SetHostnameResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetScopes forwards the call to dev.CallMethod() then parses the payload of the reply as a SetScopesResponse.
func Call_SetScopes(ctx context.Context, dev *networking.Client, request SetScopes) (SetScopesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetScopesResponse SetScopesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.SetScopesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetScopes")
		return reply.Body.SetScopesResponse, err
	}
}

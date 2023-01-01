// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_RemoveScopes forwards the call to dev.CallMethod() then parses the payload of the reply as a RemoveScopesResponse.
func Call_RemoveScopes(ctx context.Context, dev *networking.Client, request RemoveScopes) (RemoveScopesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			RemoveScopesResponse RemoveScopesResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.RemoveScopesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "RemoveScopes")
		return reply.Body.RemoveScopesResponse, err
	}
}

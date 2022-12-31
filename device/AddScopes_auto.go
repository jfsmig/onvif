// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_AddScopes forwards the call to dev.CallMethod() then parses the payload of the reply as a AddScopesResponse.
func Call_AddScopes(ctx context.Context, dev *networking.Client, request AddScopes) (AddScopesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			AddScopesResponse AddScopesResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.AddScopesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "AddScopes")
		return reply.Body.AddScopesResponse, err
	}
}

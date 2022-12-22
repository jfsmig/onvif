// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/juju/errors"
	"github.com/use-go/onvif/networking"
)

// Call_GetScopes forwards the call to dev.CallMethod() then parses the payload of the reply as a GetScopesResponse.
func Call_GetScopes(ctx context.Context, dev *networking.Client, request GetScopes) (GetScopesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetScopesResponse GetScopesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetScopesResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetScopes")
		return reply.Body.GetScopesResponse, errors.Annotate(err, "reply")
	}
}

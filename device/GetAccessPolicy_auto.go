// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetAccessPolicy forwards the call to dev.CallMethod() then parses the payload of the reply as a GetAccessPolicyResponse.
func Call_GetAccessPolicy(ctx context.Context, dev *networking.Client, request GetAccessPolicy) (GetAccessPolicyResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetAccessPolicyResponse GetAccessPolicyResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetAccessPolicyResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetAccessPolicy")
		return reply.Body.GetAccessPolicyResponse, err
	}
}

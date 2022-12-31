// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetEndpointReference forwards the call to dev.CallMethod() then parses the payload of the reply as a GetEndpointReferenceResponse.
func Call_GetEndpointReference(ctx context.Context, dev *networking.Client, request GetEndpointReference) (GetEndpointReferenceResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetEndpointReferenceResponse GetEndpointReferenceResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetEndpointReferenceResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetEndpointReference")
		return reply.Body.GetEndpointReferenceResponse, err
	}
}

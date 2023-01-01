// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetDiscoveryMode forwards the call to dev.CallMethod() then parses the payload of the reply as a GetDiscoveryModeResponse.
func Call_GetDiscoveryMode(ctx context.Context, dev *networking.Client, request GetDiscoveryMode) (GetDiscoveryModeResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetDiscoveryModeResponse GetDiscoveryModeResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetDiscoveryModeResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetDiscoveryMode")
		return reply.Body.GetDiscoveryModeResponse, err
	}
}

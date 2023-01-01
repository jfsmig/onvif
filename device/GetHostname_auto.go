// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetHostname forwards the call to dev.CallMethod() then parses the payload of the reply as a GetHostnameResponse.
func Call_GetHostname(ctx context.Context, dev *networking.Client, request GetHostname) (GetHostnameResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetHostnameResponse GetHostnameResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetHostnameResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetHostname")
		return reply.Body.GetHostnameResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetDot11Capabilities forwards the call to dev.CallMethod() then parses the payload of the reply as a GetDot11CapabilitiesResponse.
func Call_GetDot11Capabilities(ctx context.Context, dev *networking.Client, request GetDot11Capabilities) (GetDot11CapabilitiesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetDot11CapabilitiesResponse GetDot11CapabilitiesResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetDot11CapabilitiesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetDot11Capabilities")
		return reply.Body.GetDot11CapabilitiesResponse, err
	}
}

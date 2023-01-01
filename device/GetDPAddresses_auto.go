// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetDPAddresses forwards the call to dev.CallMethod() then parses the payload of the reply as a GetDPAddressesResponse.
func Call_GetDPAddresses(ctx context.Context, dev *networking.Client, request GetDPAddresses) (GetDPAddressesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetDPAddressesResponse GetDPAddressesResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetDPAddressesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetDPAddresses")
		return reply.Body.GetDPAddressesResponse, err
	}
}

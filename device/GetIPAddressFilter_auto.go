// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetIPAddressFilter forwards the call to dev.CallMethod() then parses the payload of the reply as a GetIPAddressFilterResponse.
func Call_GetIPAddressFilter(ctx context.Context, dev *networking.Client, request GetIPAddressFilter) (GetIPAddressFilterResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetIPAddressFilterResponse GetIPAddressFilterResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetIPAddressFilterResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetIPAddressFilter")
		return reply.Body.GetIPAddressFilterResponse, err
	}
}

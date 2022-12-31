// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetIPAddressFilter forwards the call to dev.CallMethod() then parses the payload of the reply as a SetIPAddressFilterResponse.
func Call_SetIPAddressFilter(ctx context.Context, dev *networking.Client, request SetIPAddressFilter) (SetIPAddressFilterResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetIPAddressFilterResponse SetIPAddressFilterResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SetIPAddressFilterResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetIPAddressFilter")
		return reply.Body.SetIPAddressFilterResponse, err
	}
}

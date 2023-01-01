// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetZeroConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a GetZeroConfigurationResponse.
func Call_GetZeroConfiguration(ctx context.Context, dev *networking.Client, request GetZeroConfiguration) (GetZeroConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetZeroConfigurationResponse GetZeroConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetZeroConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetZeroConfiguration")
		return reply.Body.GetZeroConfigurationResponse, err
	}
}

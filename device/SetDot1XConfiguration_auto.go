// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetDot1XConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a SetDot1XConfigurationResponse.
func Call_SetDot1XConfiguration(ctx context.Context, dev *networking.Client, request SetDot1XConfiguration) (SetDot1XConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetDot1XConfigurationResponse SetDot1XConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.SetDot1XConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetDot1XConfiguration")
		return reply.Body.SetDot1XConfigurationResponse, err
	}
}

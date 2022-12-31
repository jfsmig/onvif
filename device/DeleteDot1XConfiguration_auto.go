// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_DeleteDot1XConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a DeleteDot1XConfigurationResponse.
func Call_DeleteDot1XConfiguration(ctx context.Context, dev *networking.Client, request DeleteDot1XConfiguration) (DeleteDot1XConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			DeleteDot1XConfigurationResponse DeleteDot1XConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.DeleteDot1XConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "DeleteDot1XConfiguration")
		return reply.Body.DeleteDot1XConfigurationResponse, err
	}
}

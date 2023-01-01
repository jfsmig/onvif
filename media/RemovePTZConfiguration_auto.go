// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_RemovePTZConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a RemovePTZConfigurationResponse.
func Call_RemovePTZConfiguration(ctx context.Context, dev *networking.Client, request RemovePTZConfiguration) (RemovePTZConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			RemovePTZConfigurationResponse RemovePTZConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.RemovePTZConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "RemovePTZConfiguration")
		return reply.Body.RemovePTZConfigurationResponse, err
	}
}

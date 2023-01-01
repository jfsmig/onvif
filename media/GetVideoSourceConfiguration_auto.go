// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetVideoSourceConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a GetVideoSourceConfigurationResponse.
func Call_GetVideoSourceConfiguration(ctx context.Context, dev *networking.Client, request GetVideoSourceConfiguration) (GetVideoSourceConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetVideoSourceConfigurationResponse GetVideoSourceConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetVideoSourceConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetVideoSourceConfiguration")
		return reply.Body.GetVideoSourceConfigurationResponse, err
	}
}

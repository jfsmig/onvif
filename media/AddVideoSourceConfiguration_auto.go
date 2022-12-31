// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_AddVideoSourceConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a AddVideoSourceConfigurationResponse.
func Call_AddVideoSourceConfiguration(ctx context.Context, dev *networking.Client, request AddVideoSourceConfiguration) (AddVideoSourceConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			AddVideoSourceConfigurationResponse AddVideoSourceConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.AddVideoSourceConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "AddVideoSourceConfiguration")
		return reply.Body.AddVideoSourceConfigurationResponse, err
	}
}

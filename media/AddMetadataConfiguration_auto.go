// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_AddMetadataConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a AddMetadataConfigurationResponse.
func Call_AddMetadataConfiguration(ctx context.Context, dev *networking.Client, request AddMetadataConfiguration) (AddMetadataConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			AddMetadataConfigurationResponse AddMetadataConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.AddMetadataConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "AddMetadataConfiguration")
		return reply.Body.AddMetadataConfigurationResponse, err
	}
}

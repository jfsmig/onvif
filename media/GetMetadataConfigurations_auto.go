// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetMetadataConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetMetadataConfigurationsResponse.
func Call_GetMetadataConfigurations(ctx context.Context, dev *networking.Client, request GetMetadataConfigurations) (GetMetadataConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetMetadataConfigurationsResponse GetMetadataConfigurationsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetMetadataConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetMetadataConfigurations")
		return reply.Body.GetMetadataConfigurationsResponse, err
	}
}

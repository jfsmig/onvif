// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetStorageConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetStorageConfigurationsResponse.
func Call_GetStorageConfigurations(ctx context.Context, dev *networking.Client, request GetStorageConfigurations) (GetStorageConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetStorageConfigurationsResponse GetStorageConfigurationsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetStorageConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetStorageConfigurations")
		return reply.Body.GetStorageConfigurationsResponse, err
	}
}

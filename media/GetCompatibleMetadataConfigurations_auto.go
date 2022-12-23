// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetCompatibleMetadataConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCompatibleMetadataConfigurationsResponse.
func Call_GetCompatibleMetadataConfigurations(ctx context.Context, dev *networking.Client, request GetCompatibleMetadataConfigurations) (GetCompatibleMetadataConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCompatibleMetadataConfigurationsResponse GetCompatibleMetadataConfigurationsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetCompatibleMetadataConfigurationsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCompatibleMetadataConfigurations")
		return reply.Body.GetCompatibleMetadataConfigurationsResponse, err
	}
}

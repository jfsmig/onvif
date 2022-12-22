// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/juju/errors"
	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetMetadataConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetMetadataConfigurationsResponse.
func Call_GetMetadataConfigurations(ctx context.Context, dev *device.Device, request GetMetadataConfigurations) (GetMetadataConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetMetadataConfigurationsResponse GetMetadataConfigurationsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetMetadataConfigurationsResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetMetadataConfigurations")
		return reply.Body.GetMetadataConfigurationsResponse, errors.Annotate(err, "reply")
	}
}

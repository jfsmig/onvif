// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package ptz

import (
	"context"
	"github.com/juju/errors"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetCompatibleConfigurations forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCompatibleConfigurationsResponse.
func Call_GetCompatibleConfigurations(ctx context.Context, dev *networking.Client, request GetCompatibleConfigurations) (GetCompatibleConfigurationsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCompatibleConfigurationsResponse GetCompatibleConfigurationsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetCompatibleConfigurationsResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCompatibleConfigurations")
		return reply.Body.GetCompatibleConfigurationsResponse, errors.Annotate(err, "reply")
	}
}

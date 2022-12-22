// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/juju/errors"
	"github.com/use-go/onvif/networking"
)

// Call_GetGuaranteedNumberOfVideoEncoderInstances forwards the call to dev.CallMethod() then parses the payload of the reply as a GetGuaranteedNumberOfVideoEncoderInstancesResponse.
func Call_GetGuaranteedNumberOfVideoEncoderInstances(ctx context.Context, dev *networking.Client, request GetGuaranteedNumberOfVideoEncoderInstances) (GetGuaranteedNumberOfVideoEncoderInstancesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetGuaranteedNumberOfVideoEncoderInstancesResponse GetGuaranteedNumberOfVideoEncoderInstancesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetGuaranteedNumberOfVideoEncoderInstancesResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetGuaranteedNumberOfVideoEncoderInstances")
		return reply.Body.GetGuaranteedNumberOfVideoEncoderInstancesResponse, errors.Annotate(err, "reply")
	}
}

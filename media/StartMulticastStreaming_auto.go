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

// Call_StartMulticastStreaming forwards the call to dev.CallMethod() then parses the payload of the reply as a StartMulticastStreamingResponse.
func Call_StartMulticastStreaming(ctx context.Context, dev *device.Device, request StartMulticastStreaming) (StartMulticastStreamingResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			StartMulticastStreamingResponse StartMulticastStreamingResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.StartMulticastStreamingResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "StartMulticastStreaming")
		return reply.Body.StartMulticastStreamingResponse, errors.Annotate(err, "reply")
	}
}

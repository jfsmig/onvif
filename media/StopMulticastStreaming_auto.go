// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/juju/errors"
	"github.com/jfsmig/onvif/networking"
)

// Call_StopMulticastStreaming forwards the call to dev.CallMethod() then parses the payload of the reply as a StopMulticastStreamingResponse.
func Call_StopMulticastStreaming(ctx context.Context, dev *networking.Client, request StopMulticastStreaming) (StopMulticastStreamingResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			StopMulticastStreamingResponse StopMulticastStreamingResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.StopMulticastStreamingResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "StopMulticastStreaming")
		return reply.Body.StopMulticastStreamingResponse, errors.Annotate(err, "reply")
	}
}

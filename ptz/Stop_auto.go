// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package ptz

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_Stop forwards the call to dev.CallMethod() then parses the payload of the reply as a StopResponse.
func Call_Stop(ctx context.Context, dev *networking.Client, request Stop) (StopResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			StopResponse StopResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.StopResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "Stop")
		return reply.Body.StopResponse, err
	}
}

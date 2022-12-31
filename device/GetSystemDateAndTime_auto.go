// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetSystemDateAndTime forwards the call to dev.CallMethod() then parses the payload of the reply as a GetSystemDateAndTimeResponse.
func Call_GetSystemDateAndTime(ctx context.Context, dev *networking.Client, request GetSystemDateAndTime) (GetSystemDateAndTimeResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetSystemDateAndTimeResponse GetSystemDateAndTimeResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetSystemDateAndTimeResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetSystemDateAndTime")
		return reply.Body.GetSystemDateAndTimeResponse, err
	}
}

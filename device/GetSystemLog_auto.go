// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetSystemLog forwards the call to dev.CallMethod() then parses the payload of the reply as a GetSystemLogResponse.
func Call_GetSystemLog(ctx context.Context, dev *networking.Client, request GetSystemLog) (GetSystemLogResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetSystemLogResponse GetSystemLogResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetSystemLogResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetSystemLog")
		return reply.Body.GetSystemLogResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetOSD forwards the call to dev.CallMethod() then parses the payload of the reply as a GetOSDResponse.
func Call_GetOSD(ctx context.Context, dev *networking.Client, request GetOSD) (GetOSDResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetOSDResponse GetOSDResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetOSDResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetOSD")
		return reply.Body.GetOSDResponse, err
	}
}

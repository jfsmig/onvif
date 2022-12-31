// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_DeleteOSD forwards the call to dev.CallMethod() then parses the payload of the reply as a DeleteOSDResponse.
func Call_DeleteOSD(ctx context.Context, dev *networking.Client, request DeleteOSD) (DeleteOSDResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			DeleteOSDResponse DeleteOSDResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.DeleteOSDResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "DeleteOSD")
		return reply.Body.DeleteOSDResponse, err
	}
}

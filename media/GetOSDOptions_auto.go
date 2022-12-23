// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetOSDOptions forwards the call to dev.CallMethod() then parses the payload of the reply as a GetOSDOptionsResponse.
func Call_GetOSDOptions(ctx context.Context, dev *networking.Client, request GetOSDOptions) (GetOSDOptionsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetOSDOptionsResponse GetOSDOptionsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetOSDOptionsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetOSDOptions")
		return reply.Body.GetOSDOptionsResponse, err
	}
}

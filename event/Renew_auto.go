// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package event

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_Renew forwards the call to dev.CallMethod() then parses the payload of the reply as a RenewResponse.
func Call_Renew(ctx context.Context, dev *networking.Client, request Renew) (RenewResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			RenewResponse RenewResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.RenewResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "Renew")
		return reply.Body.RenewResponse, err
	}
}

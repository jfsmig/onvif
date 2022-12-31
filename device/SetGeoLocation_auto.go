// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetGeoLocation forwards the call to dev.CallMethod() then parses the payload of the reply as a SetGeoLocationResponse.
func Call_SetGeoLocation(ctx context.Context, dev *networking.Client, request SetGeoLocation) (SetGeoLocationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetGeoLocationResponse SetGeoLocationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SetGeoLocationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetGeoLocation")
		return reply.Body.SetGeoLocationResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetGeoLocation forwards the call to dev.CallMethod() then parses the payload of the reply as a GetGeoLocationResponse.
func Call_GetGeoLocation(ctx context.Context, dev *networking.Client, request GetGeoLocation) (GetGeoLocationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetGeoLocationResponse GetGeoLocationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetGeoLocationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetGeoLocation")
		return reply.Body.GetGeoLocationResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetProfiles forwards the call to dev.CallMethod() then parses the payload of the reply as a GetProfilesResponse.
func Call_GetProfiles(ctx context.Context, dev *networking.Client, request GetProfiles) (GetProfilesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetProfilesResponse GetProfilesResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetProfilesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetProfiles")
		return reply.Body.GetProfilesResponse, err
	}
}

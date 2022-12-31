// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_CreateProfile forwards the call to dev.CallMethod() then parses the payload of the reply as a CreateProfileResponse.
func Call_CreateProfile(ctx context.Context, dev *networking.Client, request CreateProfile) (CreateProfileResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			CreateProfileResponse CreateProfileResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.CreateProfileResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "CreateProfile")
		return reply.Body.CreateProfileResponse, err
	}
}

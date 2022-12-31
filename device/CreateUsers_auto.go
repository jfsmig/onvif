// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_CreateUsers forwards the call to dev.CallMethod() then parses the payload of the reply as a CreateUsersResponse.
func Call_CreateUsers(ctx context.Context, dev *networking.Client, request CreateUsers) (CreateUsersResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			CreateUsersResponse CreateUsersResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.CreateUsersResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "CreateUsers")
		return reply.Body.CreateUsersResponse, err
	}
}

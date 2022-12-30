// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetSystemBackup forwards the call to dev.CallMethod() then parses the payload of the reply as a GetSystemBackupResponse.
func Call_GetSystemBackup(ctx context.Context, dev *networking.Client, request GetSystemBackup) (GetSystemBackupResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetSystemBackupResponse GetSystemBackupResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetSystemBackupResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetSystemBackup")
		return reply.Body.GetSystemBackupResponse, err
	}
}

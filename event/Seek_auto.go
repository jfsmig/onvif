// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package event

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_Seek forwards the call to dev.CallMethod() then parses the payload of the reply as a SeekResponse.
func Call_Seek(ctx context.Context, dev *networking.Client, request Seek) (SeekResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SeekResponse SeekResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SeekResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "Seek")
		return reply.Body.SeekResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package ptz

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetPreset forwards the call to dev.CallMethod() then parses the payload of the reply as a SetPresetResponse.
func Call_SetPreset(ctx context.Context, dev *networking.Client, request SetPreset) (SetPresetResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetPresetResponse SetPresetResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.SetPresetResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetPreset")
		return reply.Body.SetPresetResponse, err
	}
}

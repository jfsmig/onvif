// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package ptz

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_OperatePresetTour forwards the call to dev.CallMethod() then parses the payload of the reply as a OperatePresetTourResponse.
func Call_OperatePresetTour(ctx context.Context, dev *networking.Client, request OperatePresetTour) (OperatePresetTourResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			OperatePresetTourResponse OperatePresetTourResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.OperatePresetTourResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "OperatePresetTour")
		return reply.Body.OperatePresetTourResponse, err
	}
}

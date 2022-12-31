// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_StartFirmwareUpgrade forwards the call to dev.CallMethod() then parses the payload of the reply as a StartFirmwareUpgradeResponse.
func Call_StartFirmwareUpgrade(ctx context.Context, dev *networking.Client, request StartFirmwareUpgrade) (StartFirmwareUpgradeResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			StartFirmwareUpgradeResponse StartFirmwareUpgradeResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.StartFirmwareUpgradeResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "StartFirmwareUpgrade")
		return reply.Body.StartFirmwareUpgradeResponse, err
	}
}

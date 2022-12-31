// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_UpgradeSystemFirmware forwards the call to dev.CallMethod() then parses the payload of the reply as a UpgradeSystemFirmwareResponse.
func Call_UpgradeSystemFirmware(ctx context.Context, dev *networking.Client, request UpgradeSystemFirmware) (UpgradeSystemFirmwareResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			UpgradeSystemFirmwareResponse UpgradeSystemFirmwareResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.UpgradeSystemFirmwareResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "UpgradeSystemFirmware")
		return reply.Body.UpgradeSystemFirmwareResponse, err
	}
}

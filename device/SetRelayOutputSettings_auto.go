// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetRelayOutputSettings forwards the call to dev.CallMethod() then parses the payload of the reply as a SetRelayOutputSettingsResponse.
func Call_SetRelayOutputSettings(ctx context.Context, dev *networking.Client, request SetRelayOutputSettings) (SetRelayOutputSettingsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetRelayOutputSettingsResponse SetRelayOutputSettingsResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.SetRelayOutputSettingsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetRelayOutputSettings")
		return reply.Body.SetRelayOutputSettingsResponse, err
	}
}

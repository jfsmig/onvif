// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/juju/errors"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetNetworkInterfaces forwards the call to dev.CallMethod() then parses the payload of the reply as a SetNetworkInterfacesResponse.
func Call_SetNetworkInterfaces(ctx context.Context, dev *Device, request SetNetworkInterfaces) (SetNetworkInterfacesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetNetworkInterfacesResponse SetNetworkInterfacesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.SetNetworkInterfacesResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetNetworkInterfaces")
		return reply.Body.SetNetworkInterfacesResponse, errors.Annotate(err, "reply")
	}
}

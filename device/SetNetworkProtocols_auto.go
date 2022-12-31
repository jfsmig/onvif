// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetNetworkProtocols forwards the call to dev.CallMethod() then parses the payload of the reply as a SetNetworkProtocolsResponse.
func Call_SetNetworkProtocols(ctx context.Context, dev *networking.Client, request SetNetworkProtocols) (SetNetworkProtocolsResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetNetworkProtocolsResponse SetNetworkProtocolsResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SetNetworkProtocolsResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetNetworkProtocols")
		return reply.Body.SetNetworkProtocolsResponse, err
	}
}

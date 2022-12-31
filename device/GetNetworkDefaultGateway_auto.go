// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetNetworkDefaultGateway forwards the call to dev.CallMethod() then parses the payload of the reply as a GetNetworkDefaultGatewayResponse.
func Call_GetNetworkDefaultGateway(ctx context.Context, dev *networking.Client, request GetNetworkDefaultGateway) (GetNetworkDefaultGatewayResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetNetworkDefaultGatewayResponse GetNetworkDefaultGatewayResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.GetNetworkDefaultGatewayResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetNetworkDefaultGateway")
		return reply.Body.GetNetworkDefaultGatewayResponse, err
	}
}

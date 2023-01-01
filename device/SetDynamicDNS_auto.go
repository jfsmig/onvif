// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetDynamicDNS forwards the call to dev.CallMethod() then parses the payload of the reply as a SetDynamicDNSResponse.
func Call_SetDynamicDNS(ctx context.Context, dev *networking.Client, request SetDynamicDNS) (SetDynamicDNSResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetDynamicDNSResponse SetDynamicDNSResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.SetDynamicDNSResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetDynamicDNS")
		return reply.Body.SetDynamicDNSResponse, err
	}
}

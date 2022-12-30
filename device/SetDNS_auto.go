// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetDNS forwards the call to dev.CallMethod() then parses the payload of the reply as a SetDNSResponse.
func Call_SetDNS(ctx context.Context, dev *networking.Client, request SetDNS) (SetDNSResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetDNSResponse SetDNSResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SetDNSResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetDNS")
		return reply.Body.SetDNSResponse, err
	}
}

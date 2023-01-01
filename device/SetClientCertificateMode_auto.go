// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetClientCertificateMode forwards the call to dev.CallMethod() then parses the payload of the reply as a SetClientCertificateModeResponse.
func Call_SetClientCertificateMode(ctx context.Context, dev *networking.Client, request SetClientCertificateMode) (SetClientCertificateModeResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetClientCertificateModeResponse SetClientCertificateModeResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.SetClientCertificateModeResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetClientCertificateMode")
		return reply.Body.SetClientCertificateModeResponse, err
	}
}

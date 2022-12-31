// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetCertificatesStatus forwards the call to dev.CallMethod() then parses the payload of the reply as a SetCertificatesStatusResponse.
func Call_SetCertificatesStatus(ctx context.Context, dev *networking.Client, request SetCertificatesStatus) (SetCertificatesStatusResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetCertificatesStatusResponse SetCertificatesStatusResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.SetCertificatesStatusResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "SetCertificatesStatus")
		return reply.Body.SetCertificatesStatusResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_LoadCertificateWithPrivateKey forwards the call to dev.CallMethod() then parses the payload of the reply as a LoadCertificateWithPrivateKeyResponse.
func Call_LoadCertificateWithPrivateKey(ctx context.Context, dev *networking.Client, request LoadCertificateWithPrivateKey) (LoadCertificateWithPrivateKeyResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			LoadCertificateWithPrivateKeyResponse LoadCertificateWithPrivateKeyResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.LoadCertificateWithPrivateKeyResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "LoadCertificateWithPrivateKey")
		return reply.Body.LoadCertificateWithPrivateKeyResponse, err
	}
}

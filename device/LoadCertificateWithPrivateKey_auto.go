// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/juju/errors"
	"github.com/jfsmig/onvif/networking"
)

// Call_LoadCertificateWithPrivateKey forwards the call to dev.CallMethod() then parses the payload of the reply as a LoadCertificateWithPrivateKeyResponse.
func Call_LoadCertificateWithPrivateKey(ctx context.Context, dev *Device, request LoadCertificateWithPrivateKey) (LoadCertificateWithPrivateKeyResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			LoadCertificateWithPrivateKeyResponse LoadCertificateWithPrivateKeyResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.LoadCertificateWithPrivateKeyResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "LoadCertificateWithPrivateKey")
		return reply.Body.LoadCertificateWithPrivateKeyResponse, errors.Annotate(err, "reply")
	}
}

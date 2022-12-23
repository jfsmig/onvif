// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_DeleteCertificates forwards the call to dev.CallMethod() then parses the payload of the reply as a DeleteCertificatesResponse.
func Call_DeleteCertificates(ctx context.Context, dev *networking.Client, request DeleteCertificates) (DeleteCertificatesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			DeleteCertificatesResponse DeleteCertificatesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.DeleteCertificatesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "DeleteCertificates")
		return reply.Body.DeleteCertificatesResponse, err
	}
}

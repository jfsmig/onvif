// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetCACertificates forwards the call to dev.CallMethod() then parses the payload of the reply as a GetCACertificatesResponse.
func Call_GetCACertificates(ctx context.Context, dev *networking.Client, request GetCACertificates) (GetCACertificatesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetCACertificatesResponse GetCACertificatesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetCACertificatesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetCACertificates")
		return reply.Body.GetCACertificatesResponse, err
	}
}

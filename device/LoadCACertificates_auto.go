// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_LoadCACertificates forwards the call to dev.CallMethod() then parses the payload of the reply as a LoadCACertificatesResponse.
func Call_LoadCACertificates(ctx context.Context, dev *networking.Client, request LoadCACertificates) (LoadCACertificatesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			LoadCACertificatesResponse LoadCACertificatesResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.LoadCACertificatesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "LoadCACertificates")
		return reply.Body.LoadCACertificatesResponse, err
	}
}

// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/juju/errors"
	"github.com/use-go/onvif/networking"
)

// Call_GetSnapshotUri forwards the call to dev.CallMethod() then parses the payload of the reply as a GetSnapshotUriResponse.
func Call_GetSnapshotUri(ctx context.Context, dev *networking.Client, request GetSnapshotUri) (GetSnapshotUriResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetSnapshotUriResponse GetSnapshotUriResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetSnapshotUriResponse, errors.Annotate(err, "call")
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetSnapshotUri")
		return reply.Body.GetSnapshotUriResponse, errors.Annotate(err, "reply")
	}
}

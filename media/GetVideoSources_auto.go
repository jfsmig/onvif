// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetVideoSources forwards the call to dev.CallMethod() then parses the payload of the reply as a GetVideoSourcesResponse.
func Call_GetVideoSources(ctx context.Context, dev *networking.Client, request GetVideoSources) (GetVideoSourcesResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetVideoSourcesResponse GetVideoSourcesResponse
		}
	}
	var reply Envelope
	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetVideoSourcesResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetVideoSources")
		return reply.Body.GetVideoSourcesResponse, err
	}
}

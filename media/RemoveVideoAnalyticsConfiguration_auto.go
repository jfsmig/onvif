// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_RemoveVideoAnalyticsConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a RemoveVideoAnalyticsConfigurationResponse.
func Call_RemoveVideoAnalyticsConfiguration(ctx context.Context, dev *networking.Client, request RemoveVideoAnalyticsConfiguration) (RemoveVideoAnalyticsConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			RemoveVideoAnalyticsConfigurationResponse RemoveVideoAnalyticsConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.RemoveVideoAnalyticsConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "RemoveVideoAnalyticsConfiguration")
		return reply.Body.RemoveVideoAnalyticsConfigurationResponse, err
	}
}

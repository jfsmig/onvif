// Code generated : DO NOT EDIT.
// Copyright (c) 2022 Jean-Francois SMIGIELSKI
// Distributed under the MIT License

package media

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_AddVideoAnalyticsConfiguration forwards the call to dev.CallMethod() then parses the payload of the reply as a AddVideoAnalyticsConfigurationResponse.
func Call_AddVideoAnalyticsConfiguration(ctx context.Context, dev *networking.Client, request AddVideoAnalyticsConfiguration) (AddVideoAnalyticsConfigurationResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			AddVideoAnalyticsConfigurationResponse AddVideoAnalyticsConfigurationResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	defer httpReply.Body.Close()
	if err != nil {
		return reply.Body.AddVideoAnalyticsConfigurationResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "AddVideoAnalyticsConfiguration")
		return reply.Body.AddVideoAnalyticsConfigurationResponse, err
	}
}

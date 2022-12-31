package sdk

import (
	"context"

	"github.com/jfsmig/onvif/event"
)

type Event struct {
	Capabilities event.Capabilities
	Properties   event.GetEventPropertiesResponse
}

func (dw *deviceWrapper) FetchEvent(ctx context.Context) Event {
	out := Event{}

	if capa, err := event.Call_GetServiceCapabilities(ctx, dw.client, event.GetServiceCapabilities{}); err == nil {
		out.Capabilities = capa.Capabilities
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("event")
	}

	if props, err := event.Call_GetEventProperties(ctx, dw.client, event.GetEventProperties{}); err == nil {
		out.Properties = props
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetEventProperties").Msg("event")
	}

	return out
}

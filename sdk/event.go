// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

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

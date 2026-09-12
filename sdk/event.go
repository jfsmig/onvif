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
	"sync"

	"github.com/jfsmig/onvif/event"
)

type Event struct {
	// A pointer, so that a failed GetServiceCapabilities is null in a dump rather than a
	// struct of false bools, which reads as a camera that supports nothing. Same shape as
	// DeviceDescriptor.Capabilities, and the same reason.
	Capabilities *event.Capabilities
	Properties   event.GetEventPropertiesResponse
}

// FetchEvent issues its two operations concurrently.
//
// Each closure writes a different field of out, which is what makes the fan-out safe
// without a mutex, and it is the same argument the fan-outs in profiles.go rest on. Not
// errgroup: a camera without the events service errors on both calls, and
// first-error-cancels would turn a partial result into an empty one.
func (p *ProfileS) FetchEvent(ctx context.Context) Event {
	out := Event{}

	var wg sync.WaitGroup

	wg.Go(func() {
		if capa, err := event.Call_GetServiceCapabilities(ctx, p.client, event.GetServiceCapabilities{}); err == nil {
			out.Capabilities = &capa.Capabilities
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("event")
		}
	})

	wg.Go(func() {
		if props, err := event.Call_GetEventProperties(ctx, p.client, event.GetEventProperties{}); err == nil {
			out.Properties = props
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetEventProperties").Msg("event")
		}
	})

	wg.Wait()
	return out
}

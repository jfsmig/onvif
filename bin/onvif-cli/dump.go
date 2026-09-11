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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
	"github.com/spf13/cobra"
)

// targetHelp is the paragraph every command taking a TARGET shows -- every dump leaf, and
// `subscribe`. The argument is the one thing nobody guesses, and the identifier form has two
// properties an operator has to be told about: it costs a discovery round, and it is the
// only form that can select a per-camera credentials file.
const targetHelp = `TARGET is either the camera's address, the XADDR column of ` + "`onvif-cli discover`" + `,
or its WS-Discovery identifier, the UUID column.

The identifier form probes the LAN to find the address, so it takes a few seconds longer
and finds only a camera that answers discovery on a non-virtual interface. It is also the
only form that selects a per-camera credentials file: an address carries no identifier, so
a camera named that way always gets the blanket credentials. A host whose name happens to
be spelled like a UUID is told apart by adding the port.`

// dumpCommand builds one leaf of `dump`. They all take the same single positional and
// differ only in what they print, so the argument handling -- which has to tell an address
// from an identifier, and probe the LAN for the latter -- is written once rather than once
// per leaf.
func dumpCommand(ctx context.Context, use string, aliases []string, short string,
	run func(context.Context, networking.ClientInfo) error) *cobra.Command {
	return &cobra.Command{
		Use:     use + " TARGET",
		Aliases: aliases,
		Short:   short,
		Long:    short + ".\n\n" + targetHelp,
		Example: "  onvif-cli dump " + use + " 192.168.1.70:80\n" +
			"  onvif-cli dump " + use + " urn:uuid:00000700-0013-0008-0203-ec71db76e907",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOneShot(ctx, func(ctx context.Context) error {
				info, err := resolveTarget(ctx, args[0])
				if err != nil {
					return err
				}
				return run(ctx, info)
			})
		},
	}
}

type OnvifFullOutput struct {
	Descriptor     sdk.DeviceDescriptor
	DeviceSystem   sdk.DeviceSystem
	DeviceSecurity sdk.DeviceSecurity
	DeviceNetwork  sdk.DeviceNetwork
	Media          sdk.Media
	Ptz            sdk.Ptz
	Profiles       sdk.MediaProfiles
	Events         sdk.Event
}

type OnvifDeviceOutput struct {
	Descriptor     sdk.DeviceDescriptor
	DeviceSystem   sdk.DeviceSystem
	DeviceSecurity sdk.DeviceSecurity
	DeviceNetwork  sdk.DeviceNetwork
}

func dumpAll(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance, s *sdk.ProfileS) interface{} {
		out := OnvifFullOutput{}
		var wg sync.WaitGroup
		wg.Go(func() { out.Descriptor = s.FetchDeviceDescriptor(ctx) })
		wg.Go(func() { out.DeviceNetwork = s.FetchDeviceNetwork(ctx) })
		wg.Go(func() { out.DeviceSystem = s.FetchDeviceSystem(ctx) })
		wg.Go(func() { out.DeviceSecurity = s.FetchDeviceSecurity(ctx) })
		wg.Go(func() { out.Media = s.FetchMedia(ctx) })
		wg.Go(func() { out.Ptz = s.FetchPTZ(ctx) })
		wg.Go(func() { out.Events = s.FetchEvent(ctx) })
		wg.Go(func() { out.Profiles = s.FetchMediaProfiles(ctx) })
		wg.Wait()
		return out
	})
}

func dumpMedia(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance, s *sdk.ProfileS) interface{} {
		return s.FetchMedia(ctx)
	})
}

func dumpDescriptor(ctx context.Context, params networking.ClientInfo) error {
	type Output struct {
		Services   map[string]string
		UUID       string
		Descriptor sdk.DeviceDescriptor
	}
	return dumpSomething(ctx, params, func(app sdk.Appliance, s *sdk.ProfileS) interface{} {
		out := Output{}
		out.UUID = app.GetUUID()
		out.Services = app.GetServices()
		out.Descriptor = s.FetchDeviceDescriptor(ctx)
		return out
	})
}

func dumpPTZ(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance, s *sdk.ProfileS) interface{} {
		return s.FetchPTZ(ctx)
	})
}

func dumpEvents(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance, s *sdk.ProfileS) interface{} {
		return s.FetchEvent(ctx)
	})
}

func dumpDevice(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance, s *sdk.ProfileS) interface{} {
		out := OnvifDeviceOutput{}
		var wg sync.WaitGroup
		wg.Go(func() { out.Descriptor = s.FetchDeviceDescriptor(ctx) })
		wg.Go(func() { out.DeviceNetwork = s.FetchDeviceNetwork(ctx) })
		wg.Go(func() { out.DeviceSystem = s.FetchDeviceSystem(ctx) })
		wg.Go(func() { out.DeviceSecurity = s.FetchDeviceSecurity(ctx) })
		wg.Wait()
		return out
	})
}

func dumpProfiles(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance, s *sdk.ProfileS) interface{} {
		return s.FetchMediaProfiles(ctx)
	})
}

// ErrNoProfileS reports an appliance that does not advertise what ONVIF Profile S requires.
//
// Reported rather than shrugged off: every dump here is built from Profile S operations, so
// without that client there is nothing to print, and emitting an object full of zero values
// would look like a camera that answered with nothing rather than one that was never asked.
var ErrNoProfileS = errors.New("the appliance advertises no ONVIF Profile S services (device, media)")

func dumpSomething(ctx context.Context, params networking.ClientInfo, generate func(app sdk.Appliance, s *sdk.ProfileS) interface{}) error {
	auth, source := credentialsFor(params.Uuid)
	sdkDev, err := sdk.NewDevice(ctx, params, auth, &httpClient)
	if err != nil {
		// Naming the source turns the most confusing failure this tool produces into an
		// actionable one: a 401 answered with the compiled-in admin/admin looks exactly
		// like a protocol failure.
		return fmt.Errorf("connecting to %s with the %s credentials: %w", params.Xaddr, source, err)
	}
	profileS, ok := sdkDev.ProfileS()
	if !ok {
		return ErrNoProfileS
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(generate(sdkDev, profileS))
}

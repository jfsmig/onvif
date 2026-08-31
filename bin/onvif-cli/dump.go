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
	"os"
	"sync"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
)

type OnvifFullOutput struct {
	Descriptor     sdk.DeviceDescriptor
	DeviceSystem   sdk.DeviceSystem
	DeviceSecurity sdk.DeviceSecurity
	DeviceNetwork  sdk.DeviceNetwork
	Media          sdk.Media
	Ptz            sdk.Ptz
	Profiles       sdk.Profiles
	Events         sdk.Event
}

type OnvifDeviceOutput struct {
	Descriptor     sdk.DeviceDescriptor
	DeviceSystem   sdk.DeviceSystem
	DeviceSecurity sdk.DeviceSecurity
	DeviceNetwork  sdk.DeviceNetwork
}

func dumpAll(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance) interface{} {
		out := OnvifFullOutput{}
		var wg sync.WaitGroup
		wg.Go(func() { out.Descriptor = app.FetchDeviceDescriptor(ctx) })
		wg.Go(func() { out.DeviceNetwork = app.FetchDeviceNetwork(ctx) })
		wg.Go(func() { out.DeviceSystem = app.FetchDeviceSystem(ctx) })
		wg.Go(func() { out.DeviceSecurity = app.FetchDeviceSecurity(ctx) })
		wg.Go(func() { out.Media = app.FetchMedia(ctx) })
		wg.Go(func() { out.Ptz = app.FetchPTZ(ctx) })
		wg.Go(func() { out.Events = app.FetchEvent(ctx) })
		wg.Go(func() { out.Profiles = app.FetchProfiles(ctx) })
		wg.Wait()
		return out
	})
}

func dumpMedia(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance) interface{} {
		return app.FetchMedia(ctx)
	})
}

func dumpDescriptor(ctx context.Context, params networking.ClientInfo) error {
	type Output struct {
		Services   map[string]string
		UUID       string
		Descriptor sdk.DeviceDescriptor
	}
	return dumpSomething(ctx, params, func(app sdk.Appliance) interface{} {
		out := Output{}
		out.UUID = app.GetUUID()
		out.Services = app.GetServices()
		out.Descriptor = app.FetchDeviceDescriptor(ctx)
		return out
	})
}

func dumpPTZ(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance) interface{} {
		return app.FetchPTZ(ctx)
	})
}

func dumpEvents(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance) interface{} {
		return app.FetchEvent(ctx)
	})
}

func dumpDevice(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance) interface{} {
		out := OnvifDeviceOutput{}
		var wg sync.WaitGroup
		wg.Go(func() { out.Descriptor = app.FetchDeviceDescriptor(ctx) })
		wg.Go(func() { out.DeviceNetwork = app.FetchDeviceNetwork(ctx) })
		wg.Go(func() { out.DeviceSystem = app.FetchDeviceSystem(ctx) })
		wg.Go(func() { out.DeviceSecurity = app.FetchDeviceSecurity(ctx) })
		wg.Wait()
		return out
	})
}

func dumpProfiles(ctx context.Context, params networking.ClientInfo) error {
	return dumpSomething(ctx, params, func(app sdk.Appliance) interface{} {
		return app.FetchProfiles(ctx)
	})
}

func dumpSomething(ctx context.Context, params networking.ClientInfo, generate func(app sdk.Appliance) interface{}) error {
	sdkDev, err := sdk.NewDevice(ctx, params, auth, &httpClient)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(generate(sdkDev))
}

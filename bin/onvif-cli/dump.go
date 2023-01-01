// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"encoding/json"
	"os"

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
		r := Runner{}
		r.Async(func() { out.Descriptor = app.FetchDeviceDescriptor(ctx) })
		r.Async(func() { out.DeviceNetwork = app.FetchDeviceNetwork(ctx) })
		r.Async(func() { out.DeviceSystem = app.FetchDeviceSystem(ctx) })
		r.Async(func() { out.DeviceSecurity = app.FetchDeviceSecurity(ctx) })
		r.Async(func() { out.Media = app.FetchMedia(ctx) })
		r.Async(func() { out.Ptz = app.FetchPTZ(ctx) })
		r.Async(func() { out.Events = app.FetchEvent(ctx) })
		r.Async(func() { out.Profiles = app.FetchProfiles(ctx) })
		r.Wait()
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
		r := Runner{}
		out.UUID = app.GetUUID()
		out.Services = app.GetServices()
		out.Descriptor = app.FetchDeviceDescriptor(ctx)
		r.Wait()
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
		r := Runner{}
		r.Async(func() { out.Descriptor = app.FetchDeviceDescriptor(ctx) })
		r.Async(func() { out.DeviceNetwork = app.FetchDeviceNetwork(ctx) })
		r.Async(func() { out.DeviceSystem = app.FetchDeviceSystem(ctx) })
		r.Async(func() { out.DeviceSecurity = app.FetchDeviceSecurity(ctx) })
		r.Wait()
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

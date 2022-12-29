// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"encoding/json"
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
	"os"
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

func dumpAll(ctx context.Context, params networking.ClientParams) error {
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return err
	}

	out := OnvifFullOutput{}

	r := Runner{}
	r.Async(func() { out.Descriptor = sdkDev.FetchDeviceDescriptor(ctx) })
	r.Async(func() { out.DeviceNetwork = sdkDev.FetchDeviceNetwork(ctx) })
	r.Async(func() { out.DeviceSystem = sdkDev.FetchDeviceSystem(ctx) })
	r.Async(func() { out.DeviceSecurity = sdkDev.FetchDeviceSecurity(ctx) })
	r.Async(func() { out.Media = sdkDev.FetchMedia(ctx) })
	r.Async(func() { out.Ptz = sdkDev.FetchPTZ(ctx) })
	r.Async(func() { out.Events = sdkDev.FetchEvent(ctx) })
	r.Async(func() { out.Profiles = sdkDev.FetchProfiles(ctx) })
	r.Wait()

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func dumpMedia(ctx context.Context, params networking.ClientParams) error {
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return err
	}

	out := sdkDev.FetchMedia(ctx)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func dumpDescriptor(ctx context.Context, params networking.ClientParams) error {
	type Output struct {
		Services   map[string]string
		UUID       string
		Descriptor sdk.DeviceDescriptor
	}
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return err
	}

	out := Output{}

	out.UUID = sdkDev.GetUUID()
	out.Services = sdkDev.GetServices()
	out.Descriptor = sdkDev.FetchDeviceDescriptor(ctx)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func dumpPTZ(ctx context.Context, params networking.ClientParams) error {
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return err
	}

	out := sdkDev.FetchPTZ(ctx)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func dumpEvents(ctx context.Context, params networking.ClientParams) error {
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return err
	}

	out := sdkDev.FetchEvent(ctx)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func dumpDevice(ctx context.Context, params networking.ClientParams) error {
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return err
	}

	out := OnvifDeviceOutput{}

	r := Runner{}
	r.Async(func() { out.Descriptor = sdkDev.FetchDeviceDescriptor(ctx) })
	r.Async(func() { out.DeviceNetwork = sdkDev.FetchDeviceNetwork(ctx) })
	r.Async(func() { out.DeviceSystem = sdkDev.FetchDeviceSystem(ctx) })
	r.Async(func() { out.DeviceSecurity = sdkDev.FetchDeviceSecurity(ctx) })
	r.Wait()

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func dumpProfiles(ctx context.Context, params networking.ClientParams) error {
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return err
	}

	out := sdkDev.FetchProfiles(ctx)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

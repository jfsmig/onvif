// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"encoding/json"
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
	"github.com/juju/errors"
	"os"
)

type OnvifFullOutput struct {
	Descriptor sdk.DeviceDescriptor
	Device     DeviceOutput
	Media      sdk.Media
	Ptz        PtzOutput
}

func dumpAll(ctx context.Context, params networking.ClientParams) error {
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return errors.Trace(err)
	}

	out := OnvifFullOutput{}

	r := Runner{}
	r.Async(func() { out.Descriptor = sdkDev.FetchDescriptor(ctx) })
	//r.Async(func() { out.Device = detailDevice(ctx, dev) })
	r.Async(func() { out.Media = sdkDev.FetchMedia(ctx) })
	//r.Async(func() { out.Ptz = detailPtz(ctx, dev) })
	r.Wait()

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func dumpMedia(ctx context.Context, params networking.ClientParams) error {
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return errors.Trace(err)
	}

	out := sdkDev.FetchMedia(ctx)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func dumpDescriptor(ctx context.Context, params networking.ClientParams) error {
	sdkDev, err := sdk.NewDevice(params)
	if err != nil {
		return errors.Trace(err)
	}

	out := sdkDev.FetchDescriptor(ctx)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

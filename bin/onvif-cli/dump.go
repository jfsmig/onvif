// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"encoding/json"
	"github.com/juju/errors"
	"github.com/use-go/onvif/networking"
	"github.com/use-go/onvif/sdk"
	"os"
)

// Used once to generate the first skeleton of code
//#go:generate go run github.com/use-go/onvif/bin/onvif-codegen cli ptz    ../../ptz/calls.txt
//#go:generate go run github.com/use-go/onvif/bin/onvif-codegen cli device ../../device/calls.txt
//#go:generate go run github.com/use-go/onvif/bin/onvif-codegen cli media  ../../media/calls.txt
//#go:generate go run github.com/use-go/onvif/bin/onvif-codegen cli event  ../../event/calls.txt

type OnvifFullOutput struct {
	Endpoint   string
	Descriptor sdk.DeviceDescriptor
	Device     DeviceOutput
	Media      sdk.Media
	Ptz        PtzOutput
}

func dumpAll(ctx context.Context, endpoint string) error {
	sdkDev, err := sdk.NewDevice(networking.ClientParams{
		endpoint,
		"admin",
		"ollyhgqo",
		nil})
	if err != nil {
		return errors.Trace(err)
	}

	out := OnvifFullOutput{}
	out.Endpoint = endpoint

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

func dumpMedia(ctx context.Context, endpoint string) error {
	sdkDev, err := sdk.NewDevice(networking.ClientParams{
		endpoint,
		"admin",
		"ollyhgqo",
		nil})
	if err != nil {
		return errors.Trace(err)
	}

	out := sdkDev.FetchMedia(ctx)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func dumpDescriptor(ctx context.Context, endpoint string) error {
	sdkDev, err := sdk.NewDevice(networking.ClientParams{
		endpoint,
		"admin",
		"ollyhgqo",
		nil})
	if err != nil {
		return errors.Trace(err)
	}

	out := sdkDev.FetchDescriptor(ctx)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

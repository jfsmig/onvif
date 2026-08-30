// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"fmt"
	"net"

	"github.com/jfsmig/go-wsd/wsd"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
	"github.com/jfsmig/onvif/utils"
)

// probeOptions keeps the dialect the in-tree probe spoke: the zero Flavor is the
// April 2005 draft that ONVIF mandates. HopLimit carries over the TTL the previous
// implementation was set to.
var probeOptions = wsd.ProbeOptions{HopLimit: 4}

// itfProbe is the outcome of probing one interface, kept so the probes can run
// concurrently while the output stays in interface order.
type itfProbe struct {
	name    string
	devices []wsd.Device
	err     error
}

func discover(ctx context.Context, flagStreams bool) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	// Every probe waits out a fixed collection window, so probing in sequence made the
	// total grow with the number of interfaces: on a host carrying a dozen veth devices
	// the deadline in main() expired before the real NIC had been reached.
	probes := make([]*itfProbe, 0, len(interfaces))
	runner := utils.Runner{}
	for _, itf := range interfaces {
		if itf.Flags&net.FlagUp == 0 || itf.Flags&net.FlagLoopback != 0 {
			continue
		}
		probe := &itfProbe{name: itf.Name}
		probes = append(probes, probe)
		runner.Async(func() { probe.devices, probe.err = wsd.Discover(ctx, probe.name, probeOptions) })
	}
	runner.Wait()

	for _, probe := range probes {
		if probe.err != nil {
			Logger.Warn().Str("itf", probe.name).Err(probe.err).Msg("lan discovery failed")
			continue
		}
		Logger.Trace().Str("itf", probe.name).Int("devices", len(probe.devices)).Msg("lan discovery")
		for _, found := range probe.devices {
			dev := networking.ClientInfo{Xaddr: found.Xaddr, Uuid: found.UUID}
			if dev.Uuid == "" {
				dev.Uuid = "-"
			}
			if !flagStreams {
				fmt.Println(dev.Xaddr, dev.Uuid)
			} else {
				if cam, err := sdk.NewDevice(ctx, dev, auth, &httpClient); err != nil {
					Logger.Error().Err(err).Msg("Camera instantiation failure")
				} else {
					profiles := cam.FetchProfiles(ctx).Profiles
					for id, profile := range profiles {
						fmt.Println(dev.Xaddr, dev.Uuid, id, profile.Uris.Stream.Uri, profile.Uris.Snapshot.Uri)
					}
				}
			}
		}
	}
	return nil
}

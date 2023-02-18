// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"fmt"
	"github.com/jfsmig/onvif/sdk"
	"net"

	wsdiscovery "github.com/jfsmig/onvif/ws-discovery"
)

func discover(ctx context.Context, flagStreams bool) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, itf := range interfaces {
		devices, err := wsdiscovery.GetAvailableDevicesAtSpecificEthernetInterface(itf.Name)
		if err != nil {
			Logger.Warn().Str("itf", itf.Name).Msg("lan discovery failed")
		} else {
			for _, dev := range devices {
				if dev.Uuid == "" {
					dev.Uuid = "-"
				}
				if !flagStreams {
					fmt.Println(dev.Xaddr, dev.Uuid)
				} else {
					if cam, err := sdk.NewDevice(ctx, dev, auth, &httpClient); err != nil {
						Logger.Error().Err(err).Msg("Camera instanciation failure")
					} else {
						for id, profile := range cam.FetchProfiles(ctx).Profiles {
							fmt.Println(dev.Xaddr, dev.Uuid, id, profile.Uris.Stream.Uri, profile.Uris.Snapshot.Uri)
						}
					}
				}
			}
		}
	}
	return nil
}

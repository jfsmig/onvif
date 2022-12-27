// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"fmt"
	wsdiscovery "github.com/jfsmig/onvif/ws-discovery"
	"net"
	"net/url"
)

func discover(ctx context.Context) error {
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
				u := dev.GetEndpoint("device")
				parsedUrl, err := url.Parse(u)
				if err != nil {
					Logger.Warn().Err(err).Str("action", "parse").Msg("invalid device")
				} else {
					uuid := dev.GetUUID()
					if uuid == "" {
						uuid = "-"
					}
					fmt.Println(parsedUrl.Host, uuid)
				}
			}
		}
	}
	return nil
}

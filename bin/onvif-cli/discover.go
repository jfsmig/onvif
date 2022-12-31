// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"fmt"
	"net"

	wsdiscovery "github.com/jfsmig/onvif/ws-discovery"
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
				if dev.Uuid == "" {
					dev.Uuid = "-"
				}
				fmt.Println(dev.Xaddr, dev.Uuid)
			}
		}
	}
	return nil
}

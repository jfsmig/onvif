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
	"fmt"
	"net"
	"sync"

	"github.com/jfsmig/go-wsd/wsd"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
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

// discoverOptions carries the decisions the caller makes, so that none of them reaches
// the callee as a bare boolean at the call site. The zero value discovers the cameras on
// the interfaces a camera can plausibly be reached through.
type discoverOptions struct {
	streams       bool // report each device's media profiles, not just its address
	allInterfaces bool // keep the container, VM and overlay devices in the probe set
}

func discover(ctx context.Context, opts discoverOptions) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	probeable, skipped := probeableInterfaceNames(interfaces, opts.allInterfaces)
	if len(probeable) == 0 {
		warnNothingToProbe(skipped)
		return nil
	}

	// The probe set by name and the rest by count. zerolog's global level is trace, so
	// this line prints on every run, and on a host running containers the skipped names
	// are a dozen and a half of veth noise on one line. The count still says a filter ran,
	// and --all names what it took. Above the fan-out, so that the line cannot be written
	// after a probe has already finished.
	Logger.Trace().Strs("probing", probeable).Int("skipped", len(skipped)).Msg("lan discovery started")

	// Every probe waits out a fixed collection window, so probing in sequence made the
	// total grow with the number of interfaces: on a host carrying a dozen veth devices
	// the deadline in main() expired before the real NIC had been reached.
	probes := make([]*itfProbe, 0, len(probeable))
	var wg sync.WaitGroup
	for _, name := range probeable {
		probe := &itfProbe{name: name}
		probes = append(probes, probe)
		wg.Go(func() { probe.devices, probe.err = wsd.Discover(ctx, probe.name, probeOptions) })
	}
	wg.Wait()
	Logger.Trace().Int("interfaces", len(probes)).Msg("lan discovery ended")

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
			if !opts.streams {
				fmt.Println(probe.name, dev.Xaddr, dev.Uuid)
			} else {
				if cam, err := sdk.NewDevice(ctx, dev, auth, &httpClient); err != nil {
					Logger.Error().Str("itf", probe.name).Str("addr", dev.Xaddr).Str("uuid", dev.Uuid).Err(err).
						Msg("Camera instantiation failure")
				} else if profileS, ok := cam.ProfileS(); !ok {
					Logger.Warn().Str("itf", probe.name).Str("addr", dev.Xaddr).Str("uuid", dev.Uuid).
						Msg("No ONVIF Profile S service advertised, no stream to report")
				} else {
					profiles := profileS.FetchMediaProfiles(ctx).Profiles
					for id, profile := range profiles {
						fmt.Println(probe.name, dev.Xaddr, dev.Uuid, id, profile.Uris.Stream.Uri, profile.Uris.Snapshot.Uri)
					}
				}
			}
		}
	}
	return nil
}

// warnNothingToProbe tells the operator which of the two empty cases happened, because
// only the filtered one has a way around it: recommending --all to someone whose sole
// interface is the loopback would be advice that cannot work.
//
// Deliberately not a fallback to probing everything. The filter has to mean what it says,
// and falling open would make the probe set depend on the host's state — one interface
// coming up would silently switch the run from every virtual device on the host to that
// one interface.
func warnNothingToProbe(skipped []string) {
	if len(skipped) == 0 {
		Logger.Warn().Msg("no interface to probe, none is up and non-loopback")
		return
	}
	Logger.Warn().Strs("skipped", skipped).
		Msg("no interface left to probe, pass --all to probe the filtered ones too")
}

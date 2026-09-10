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

	"github.com/jfsmig/onvif/credentials"
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

// probeLAN runs one WS-Discovery probe per probeable interface and returns the outcome of
// each, in the order net.Interfaces() reported them. It is the whole of the discovery
// mechanics: `discover` and `streams` print what it found, and the urn:uuid: form of
// `dump` searches it for one device.
//
// Every probe waits out a fixed collection window, so probing in sequence made the total
// grow with the number of interfaces: on a host carrying a dozen veth devices the deadline
// in main() expired before the real NIC had been reached. The concurrency is therefore not
// an optimisation.
//
// An empty probe set is not an error. It yields no probes, having said on stderr which of
// the two empty cases happened, and the caller reports nothing found in its own terms.
func probeLAN(ctx context.Context, allInterfaces bool) ([]*itfProbe, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	probeable, skipped := probeableInterfaceNames(interfaces, allInterfaces)
	if len(probeable) == 0 {
		warnNothingToProbe(skipped)
		return nil, nil
	}

	// The probe set by name and the rest by count: on a host running containers the
	// skipped names are a dozen and a half of veth noise on one line, while the count
	// still says a filter ran and --all names what it took.
	//
	// At info, so that one -v says what the tool is doing during the seconds it spends
	// waiting out the collection window. Above the fan-out, so the line cannot be written
	// after a probe has already finished.
	Logger.Info().Strs("probing", probeable).Int("skipped", len(skipped)).Msg("lan discovery started")

	probes := fanOutProbes(ctx, probeable, func(ctx context.Context, itf string) ([]wsd.Device, error) {
		return wsd.Discover(ctx, itf, probeOptions)
	})
	Logger.Trace().Int("interfaces", len(probes)).Msg("lan discovery ended")

	return probes, nil
}

// prober runs one interface's probe. wsd.Discover is the only implementation; the seam
// exists so that the fan-out below can be exercised under -race without a network
// interface, which is what go-wsd itself does with its own prober type and for the same
// reason -- otherwise the only concurrency in this tool is the one thing CI never runs.
type prober func(ctx context.Context, itf string) ([]wsd.Device, error)

// fanOutProbes probes every named interface at once and returns the outcomes in the order
// the names were given.
//
// Each goroutine owns one *itfProbe and writes only that one, which is why there is no
// lock: probe is a fresh pointer per iteration and the appends are all done here. wg.Wait
// is the only thing ordering those writes against the caller's reads, so a variant that
// returned early -- on the first match, say -- would be a data race with nowhere to put a
// lock.
func fanOutProbes(ctx context.Context, names []string, send prober) []*itfProbe {
	probes := make([]*itfProbe, 0, len(names))
	var wg sync.WaitGroup
	for _, name := range names {
		probe := &itfProbe{name: name}
		probes = append(probes, probe)
		wg.Go(func() { probe.devices, probe.err = send(ctx, probe.name) })
	}
	wg.Wait()
	return probes
}

func discover(ctx context.Context, opts discoverOptions) error {
	probes, err := probeLAN(ctx, opts.allInterfaces)
	if err != nil {
		return err
	}

	for _, probe := range probes {
		if probe.err != nil {
			Logger.Warn().Str("itf", probe.name).Err(probe.err).Msg("lan discovery failed")
			continue
		}
		Logger.Trace().Str("itf", probe.name).Int("devices", len(probe.devices)).Msg("lan discovery")
		for _, found := range probe.devices {
			// The true identifier travels, and the placeholder is applied only where the
			// line is printed: a device that reported none would otherwise be looked up
			// under the name "-".
			dev := networking.ClientInfo{Xaddr: found.Xaddr, Uuid: found.UUID}
			if !opts.streams {
				fmt.Println(probe.name, dev.Xaddr, uuidColumn(dev.Uuid))
				continue
			}
			printStreams(ctx, probe.name, dev)
		}
	}
	return nil
}

// printStreams prints one line per media profile of one camera.
//
// This is where the credentials become per-camera: the identifier discovery reported keys
// the lookup, and a camera no file names still gets the blanket credentials, which is what
// a set ONVIF_USERNAME means. What a device claims is not canonicalised here -- the store
// does that on both sides of the comparison, and never rejects an identifier a device may
// legitimately have spelled its own way.
func printStreams(ctx context.Context, itf string, dev networking.ClientInfo) {
	auth, source := credentialsFor(dev.Uuid)
	cam, err := sdk.NewDevice(ctx, dev, auth, &httpClient)
	if err != nil {
		Logger.Error().Str("itf", itf).Str("addr", dev.Xaddr).Str("uuid", uuidColumn(dev.Uuid)).
			Str("credentials", source).Err(err).Msg("Camera instantiation failure")
		return
	}
	profileS, ok := cam.ProfileS()
	if !ok {
		Logger.Warn().Str("itf", itf).Str("addr", dev.Xaddr).Str("uuid", uuidColumn(dev.Uuid)).
			Msg("No ONVIF Profile S service advertised, no stream to report")
		return
	}
	for id, profile := range profileS.FetchMediaProfiles(ctx).Profiles {
		fmt.Println(itf, dev.Xaddr, uuidColumn(dev.Uuid), id,
			profile.Uris.Stream.Uri, profile.Uris.Snapshot.Uri)
	}
}

// resolveTarget turns the single argument of a dump sub-command into the parameters of a
// client: an address is taken as it stands, an identifier is looked up on the LAN.
func resolveTarget(ctx context.Context, arg string) (networking.ClientInfo, error) {
	target, err := parseDeviceTarget(arg)
	if err != nil {
		return networking.ClientInfo{}, err
	}
	if target.Xaddr != "" {
		return networking.ClientInfo{Xaddr: target.Xaddr}, nil
	}
	return lookupByUUID(ctx, target.Uuid)
}

// lookupByUUID finds the address of the device that claims an identifier.
//
// It probes the default interface set, not everything --all would take. The lookup is on
// the critical path of a command that still has its real work to do inside the one-minute
// context in main(), a probe costs a full collection window, and a camera reachable for a
// dump sits on a real link by construction. An operator who needs the wider set already
// has the better route: `discover -a`, then dump the address it printed.
//
// A device that answered on several interfaces is not an error -- a multi-homed host is
// legitimate -- so the first in interface order is used and the rest are named. There is
// no case for several addresses of one device: go-wsd keeps only the first advertised
// address, citing the ONVIF Application Programmer's Guide, so nothing here can choose.
func lookupByUUID(ctx context.Context, uuid string) (networking.ClientInfo, error) {
	// One line, at a level that prints, because the probe is several seconds of silence in
	// a command an operator expects to answer at once.
	Logger.Info().Str("uuid", uuid).Msg("resolving the identifier by WS-Discovery")

	probes, err := probeLAN(ctx, false)
	if err != nil {
		return networking.ClientInfo{}, err
	}

	var matches []networking.ClientInfo
	answered := 0
	for _, probe := range probes {
		if probe.err != nil {
			Logger.Warn().Str("itf", probe.name).Err(probe.err).Msg("lan discovery failed")
			continue
		}
		for _, found := range probe.devices {
			answered++
			// Both sides, and for different reasons: what the device claims arrives in
			// whatever spelling its firmware chose, and lookupByUUID is also reachable
			// with an identifier that has not been through parseDeviceTarget.
			if credentials.CanonicalID(found.UUID) == credentials.CanonicalID(uuid) {
				matches = append(matches, networking.ClientInfo{Xaddr: found.Xaddr, Uuid: found.UUID})
			}
		}
	}

	switch {
	case len(probes) == 0:
		// probeLAN has already said on stderr which of the two empty cases happened and
		// what to do about it; repeating "check the link" here would be advice that cannot
		// work when the interfaces were filtered rather than absent.
		return networking.ClientInfo{}, fmt.Errorf(
			"no interface was probed, so no camera could be found by its identifier; pass the camera's IP:PORT instead")
	case len(matches) == 0 && answered == 0:
		// Three distinct messages, because "nothing was probed", "the link is dead" and
		// "your identifier is wrong" have three different next actions.
		return networking.ClientInfo{}, fmt.Errorf(
			"no device answered discovery on %d interface(s); check the link, or pass the camera's IP:PORT",
			len(probes))
	case len(matches) == 0:
		return networking.ClientInfo{}, fmt.Errorf(
			"no device with identifier %s among the %d that answered; run `onvif-cli discover` to list them",
			uuid, answered)
	case len(matches) > 1:
		addrs := make([]string, 0, len(matches))
		for _, m := range matches {
			addrs = append(addrs, m.Xaddr)
		}
		Logger.Warn().Str("uuid", uuid).Strs("addr", addrs).Str("using", matches[0].Xaddr).
			Msg("the identifier answered on several links; pass IP:PORT to choose")
	}
	return matches[0], nil
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

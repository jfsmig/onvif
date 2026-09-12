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
	"maps"
	"net"
	"slices"
	"sync"

	"github.com/jfsmig/go-wsd/wsd"

	"github.com/jfsmig/onvif/v2/credentials"
	"github.com/jfsmig/onvif/v2/networking"
	"github.com/jfsmig/onvif/v2/sdk"
)

// probeOptions keeps the dialect the in-tree probe spoke: the zero Flavor is the
// April 2005 draft that ONVIF mandates. HopLimit carries over the TTL the previous
// implementation was set to.
var probeOptions = wsd.ProbeOptions{HopLimit: 4}

// foundDevice is one device a probe reported, carrying the interface it answered on so the
// output keeps that column after the cameras are queried out of order.
type foundDevice struct {
	itf string
	dev networking.ClientInfo
}

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

	// The probes are flattened before anything is fetched or printed, so the output order is
	// settled by discovery -- interface order, then the order a device answered in -- and not
	// by whichever camera turns out to be quickest to talk to below.
	var found []foundDevice
	for _, probe := range probes {
		if probe.err != nil {
			Logger.Warn().Str("itf", probe.name).Err(probe.err).Msg("lan discovery failed")
			continue
		}
		Logger.Trace().Str("itf", probe.name).Int("devices", len(probe.devices)).Msg("lan discovery")
		for _, dev := range probe.devices {
			// The true identifier travels, and the placeholder is applied only where the
			// line is printed: a device that reported none would otherwise be looked up
			// under the name "-".
			found = append(found, foundDevice{
				itf: probe.name,
				dev: networking.ClientInfo{Xaddr: dev.Xaddr, Uuid: dev.UUID},
			})
		}
	}

	if !opts.streams {
		for _, f := range found {
			fmt.Println(f.itf, f.dev.Xaddr, uuidColumn(f.dev.Uuid))
		}
		return nil
	}

	// How large a fleet this takes is bounded by the process file-descriptor limit and by
	// nothing here, exactly as in subscribe.go's streamFleet -- and this is the command an
	// operator points at a whole LAN, so it is the one that finds the limit first. Each
	// camera costs an endpoint load and then a FetchMediaProfiles that itself fans out about
	// a dozen requests per media profile, and httpClient's MaxConnsPerHost is per host, so it
	// caps nothing across a fleet of distinct hosts. Forty cameras is a few hundred sockets
	// at once, inside the one minute runOneShot allows. Past the limit the failures read as
	// unreachable cameras, and `ulimit -n` is the answer.
	//
	// One goroutine per camera, each writing only its own slot. Every camera costs an
	// endpoint load and then a GetProfiles, and doing that in the print loop made a fleet
	// cost the sum of its cameras under a deadline the whole run shares. Same shape as
	// fanOutProbes above: the slice is sized first, nothing is appended, and wg.Wait is what
	// orders the writes against the printing below.
	lines := make([][]string, len(found))
	var wg sync.WaitGroup
	for i, f := range found {
		wg.Go(func() { lines[i] = streamLines(ctx, f.itf, f.dev) })
	}
	wg.Wait()

	silent := 0
	for _, camera := range lines {
		if len(camera) == 0 {
			silent++
		}
		for _, line := range camera {
			fmt.Print(line)
		}
	}

	// A camera that answered discovery and then reported no stream is the shape a wrong
	// password takes here: sdk swallows the per-call fault at trace level, so stdout simply
	// has one fewer line than the LAN has cameras, and nothing says which camera or why.
	// Counting them is what makes a partial failure visible at all -- a fleet of four with
	// one line printed looks exactly like a fleet of one.
	//
	// Warn rather than trace, because the tool promises that warnings and errors print
	// whatever the verbosity, while -vvv is for the per-call causes underneath.
	if silent > 0 {
		Logger.Warn().Int("cameras", len(found)).Int("silent", silent).
			Msg("Cameras answered discovery but reported no stream, re-run with -vvv for the cause")
	}
	return nil
}

// streamLines builds one line per media profile of one camera, and returns them rather than
// printing them: it runs in its own goroutine, and a camera that answered first has no claim
// on being printed first. A camera that could not be reached contributes no line, which is
// what its early returns below already meant.
//
// This is where the credentials become per-camera: the identifier discovery reported keys
// the lookup, and a camera no file names still gets the blanket credentials, which is what
// a set ONVIF_USERNAME means. What a device claims is not canonicalised here -- the store
// does that on both sides of the comparison, and never rejects an identifier a device may
// legitimately have spelled its own way.
func streamLines(ctx context.Context, itf string, dev networking.ClientInfo) []string {
	auth, source := credentialsFor(dev.Uuid)
	cam, err := sdk.NewDevice(ctx, dev, auth, &httpClient)
	if err != nil {
		Logger.Error().Str("itf", itf).Str("addr", dev.Xaddr).Str("uuid", uuidColumn(dev.Uuid)).
			Str("credentials", source).Err(err).Msg("Camera instantiation failure")
		return nil
	}
	profileS, ok := cam.ProfileS()
	if !ok {
		Logger.Warn().Str("itf", itf).Str("addr", dev.Xaddr).Str("uuid", uuidColumn(dev.Uuid)).
			Msg("No ONVIF Profile S service advertised, no stream to report")
		return nil
	}

	// Sorted, for the same reason sdk.FetchStreamURI sorts the same map: a range over it
	// reordered a camera's lines on every run, and an operator diffing two runs saw a change
	// that was not one. Sprintln rather than Sprint, because Sprint inserts no separator
	// between two strings and every column here is a string type.
	profiles := profileS.FetchMediaProfiles(ctx).Profiles

	// Same boundary dumpSomething draws, and for the same reason: FetchMediaProfiles
	// swallows a per-call fault because that is the camera's answer, but an expired deadline
	// is ours and says nothing about the camera. Without this a run that timed out
	// contributed no line and no diagnostic, which is indistinguishable from a camera that
	// genuinely has no media profile.
	if err := ctx.Err(); err != nil {
		Logger.Warn().Str("itf", itf).Str("addr", dev.Xaddr).Str("uuid", uuidColumn(dev.Uuid)).
			Err(err).Msg("Camera not fully queried before the deadline, no stream reported")
		return nil
	}

	lines := make([]string, 0, len(profiles))
	for _, id := range slices.Sorted(maps.Keys(profiles)) {
		profile := profiles[id]
		lines = append(lines, fmt.Sprintln(itf, dev.Xaddr, uuidColumn(dev.Uuid), id,
			profile.Uris.Stream.Uri, profile.Uris.Snapshot.Uri))
	}
	return lines
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
	byID, answered := probedDevices(probes)
	return resolveIdentifier(probes, byID, answered, uuid)
}

// probedDevices indexes what one probe round found, by canonical identifier, keeping every
// address a device answered on, and reports how many devices answered in all.
//
// Both sides of the comparison go through credentials.CanonicalID, for the two reasons
// resolveIdentifier's caller gives: what a device claims arrives in whatever spelling its
// firmware chose, and an identifier can reach here without having been through
// parseDeviceTarget.
//
// It is the shared half of lookupByUUID and resolveTargets. One probe answers however many
// identifiers were named, which is why `subscribe` takes its targets as positionals rather
// than one at a time -- a fleet of eight would otherwise spend eight collection windows
// resolving before it streamed anything.
func probedDevices(probes []*itfProbe) (map[string][]networking.ClientInfo, int) {
	byID := make(map[string][]networking.ClientInfo)
	answered := 0
	for _, probe := range probes {
		if probe.err != nil {
			Logger.Warn().Str("itf", probe.name).Err(probe.err).Msg("lan discovery failed")
			continue
		}
		for _, found := range probe.devices {
			answered++
			id := credentials.CanonicalID(found.UUID)
			byID[id] = append(byID[id], networking.ClientInfo{Xaddr: found.Xaddr, Uuid: found.UUID})
		}
	}
	return byID, answered
}

// resolveIdentifier picks the device claiming one identifier out of an indexed probe round.
//
// Separate from the probing so that lookupByUUID and resolveTargets answer with the same
// three errors and the same warning: "nothing was probed", "the link is dead" and "your
// identifier is wrong" have three different next actions, and a fleet command must not
// invent a fourth wording for them.
func resolveIdentifier(probes []*itfProbe, byID map[string][]networking.ClientInfo,
	answered int, uuid string) (networking.ClientInfo, error) {
	matches := byID[credentials.CanonicalID(uuid)]

	switch {
	case len(probes) == 0:
		// probeLAN has already said on stderr which of the two empty cases happened and
		// what to do about it; repeating "check the link" here would be advice that cannot
		// work when the interfaces were filtered rather than absent.
		return networking.ClientInfo{}, fmt.Errorf(
			"no interface was probed, so no camera could be found by its identifier; pass the camera's IP:PORT instead")
	case len(matches) == 0 && answered == 0:
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

// resolveTargets turns the positionals of `subscribe` into client parameters, probing the
// LAN at most once however many identifiers were named.
//
// resolveTarget probes per identifier, which is right for the single argument of a dump but
// would cost one full collection window per camera here. So the addresses pass straight
// through, and the probe happens only if an identifier was actually named.
//
// An identifier no device claims fails the whole command, before anything has streamed.
// That is `dump`'s behaviour and it is the friendlier one here too: the failure arrives at
// once, with the list of identifiers that did answer, rather than leaving an operator
// watching seven of the eight cameras they asked for and wondering which is quiet.
func resolveTargets(ctx context.Context, args []string) ([]networking.ClientInfo, error) {
	targets := make([]deviceTarget, 0, len(args))
	identifiers := 0
	for _, arg := range args {
		target, err := parseDeviceTarget(arg)
		if err != nil {
			return nil, err
		}
		if target.Uuid != "" {
			identifiers++
		}
		targets = append(targets, target)
	}

	var (
		probes   []*itfProbe
		byID     map[string][]networking.ClientInfo
		answered int
	)
	if identifiers > 0 {
		Logger.Info().Int("identifiers", identifiers).
			Msg("resolving the identifiers by WS-Discovery")

		var err error
		if probes, err = probeLAN(ctx, false); err != nil {
			return nil, err
		}
		byID, answered = probedDevices(probes)
	}

	out := make([]networking.ClientInfo, 0, len(targets))
	for _, target := range targets {
		if target.Xaddr != "" {
			out = append(out, networking.ClientInfo{Xaddr: target.Xaddr})
			continue
		}
		found, err := resolveIdentifier(probes, byID, answered, target.Uuid)
		if err != nil {
			return nil, err
		}
		out = append(out, found)
	}
	return dedupeTargets(out), nil
}

// dedupeTargets drops targets that resolved to the same camera.
//
// `discover` prints one line per interface a device answered on, so the pipeline this command
// documents -- discover, awk, xargs subscribe -- hands a dual-homed camera over twice. Two
// pull points on one camera is not something a device would refuse; it just doubles every
// event on stdout with nothing to say why, which is the kind of silent wrongness worth a few
// lines to prevent. Naming a camera twice by hand is caught by the same rule.
//
// The first occurrence keeps its position, because the order of the positionals is the
// operator's. What it does not keep is a missing identifier: an identifier is the only form
// that selects a per-camera credentials file, so a camera named both by address and by
// identifier must not lose the second to the first.
func dedupeTargets(cams []networking.ClientInfo) []networking.ClientInfo {
	out := make([]networking.ClientInfo, 0, len(cams))
	seen := make(map[string]int, len(cams))

	for _, cam := range cams {
		at, duplicate := seen[cam.Xaddr]
		if !duplicate {
			seen[cam.Xaddr] = len(out)
			out = append(out, cam)
			continue
		}
		if out[at].Uuid == "" && cam.Uuid != "" {
			out[at].Uuid = cam.Uuid
		}
		Logger.Warn().Str("addr", cam.Xaddr).Str("uuid", uuidColumn(cam.Uuid)).
			Msg("target named more than once, subscribing to it once")
	}
	return out
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

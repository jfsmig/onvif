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

// Package sdk is the layer a caller wants: NewDevice connects to a camera, loads the service
// endpoints it advertises, and hands back an Appliance offering one client per ONVIF Profile.
//
// The contract that governs almost everything here, and that surprises people: a Fetch*
// method returns a struct and no error. A sub-call that failed is logged at trace level on
// the package-level Logger and leaves its field zero, so that one operation a camera does not
// implement cannot lose a whole dump -- cameras vary wildly in what they implement, and a
// partial answer is worth more than an error. Two things follow from that and are easy to get
// wrong. A field that swallowing can leave zero is a pointer wherever its zero value would
// also be a legal answer, so null in a dump means "never answered" rather than "answered no".
// And an expired or cancelled context is not swallowed by this rule at all: it is the
// caller's own deadline and says nothing about the camera, so check ctx.Err() before treating
// a returned struct as a reading.
//
// PullPoint is the deliberate exception, and its name says so: it is a chain in which every
// link is load-bearing, so it returns errors.
package sdk

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/beevik/etree"
	"github.com/rs/zerolog"

	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/media"
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/utils"
)

//go:generate go run github.com/jfsmig/onvif/bin/onvif-codegen profile sdk ./profiles

var (
	// Logger gathers the format and the destination of the diagnostics written here, and is
	// a variable so that it can be replaced: an application wanting JSON rather than the
	// console writer, or a file rather than stderr, assigns its own.
	//
	// Ownership, stated the way networking.Client states it for its own fields: reading it is
	// safe from any goroutine -- that is what zerolog is built for -- but replacing it is a
	// plain assignment to a package variable that every fan-out goroutine reads. That belongs
	// in initialisation, before the first call. A swap while calls are in flight is a data
	// race, and no lock here can cover it: the previous wording, "can be safely used from any
	// part of the application", read as permission to do exactly that.
	Logger = zerolog.
		New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().
		Logger()
)

// Appliance is one connection to one ONVIF device.
//
// A note on the http.Client a caller supplies to NewDevice: every Fetch* method issues its
// independent operations concurrently, so one call can offer a dozen or more requests to
// the device at once, and a full dump several dozen. http.Transport defaults to no
// per-host connection limit, while embedded camera web servers accept only a handful and
// reset the rest. Set Transport.MaxConnsPerHost to something small -- bin/onvif-cli uses 4
// -- or the concurrency costs more than it saves.
//
// It carries what belongs to the connection itself -- identity and the service endpoints
// learnt at load time -- and hands out one client per ONVIF Profile. The operations live on
// those clients rather than here, so that the API follows the shape of the norm: a caller
// asks the appliance which Profiles it can speak, then works inside one.
//
// Do not implement this interface outside the package. It gains a method for every Profile
// that becomes supported, so an outside implementation would break on a minor release; build
// one with NewDevice or WrapClient instead.
type Appliance interface {
	// GetUUID return the unique identifier of the remote appliance.
	// The UUID is Usually known after the discovery.
	// TODO(jfs): determine the UUID for devices that do not requires discovery
	GetUUID() string

	GetDeviceEndpoint() string

	GetEndpoint(name string) string

	GetServices() map[string]string

	// ProfileS returns a client for the operations of ONVIF Profile S, and reports whether
	// the appliance advertises the services that Profile requires unconditionally. See
	// ProfileS for what the result does and does not assert.
	ProfileS() (*ProfileS, bool)
}

type Media struct {
	Video Video
	Audio Audio
	// A pointer, so that a failed GetServiceCapabilities is null in a dump rather than a
	// struct of false bools, which reads as a camera that supports nothing. Same shape as
	// DeviceDescriptor.Capabilities, and the same reason.
	Capabilities *media.Capabilities
}

type deviceWrapper struct {
	client *networking.Client
}

func NewDevice(ctx context.Context, info networking.ClientInfo, auth networking.ClientAuth, httpClient *http.Client) (Appliance, error) {
	client, err := networking.NewClient(info, httpClient)
	if err != nil {
		return nil, err
	}
	return WrapClient(ctx, client, auth)
}

func WrapClient(ctx context.Context, client *networking.Client, auth networking.ClientAuth) (Appliance, error) {
	dw := &deviceWrapper{client: client}
	dw.client.SetAuth(auth)
	return dw.load(ctx)
}

func (dw *deviceWrapper) load(ctx context.Context) (Appliance, error) {
	// GetSystemDateAndTime is both the liveness probe and the clock probe. It is issued
	// through the generated wrapper rather than by hand so that its reply is parsed: the
	// device's own clock is what the WS-Security Created stamp has to be expressed in, and
	// this exchange used to fetch it and close the body unread.
	//
	// This request does carry a UsernameToken, stamped in local time because no offset is
	// known yet. It still works as a bootstrap because ONVIF places GetSystemDateAndTime
	// in the pre-authentication access class, so a camera answers it whether or not the
	// token validates -- which is exactly the situation on a device whose clock is skewed
	// enough to reject every other call. If that access class ever turns out not to hold,
	// this has to become an explicitly unauthenticated exchange instead.
	dt, err := device.Call_GetSystemDateAndTime(ctx, dw.client, device.GetSystemDateAndTime{})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", utils.ErrNotOnvif, err)
	}
	dw.client.SetClockOffset(deviceClockOffset(time.Now(), dt.SystemDateAndTime))

	resp, err := dw.client.CallMethod(ctx, device.GetCapabilities{Category: "All"})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", utils.ErrNotOnvif, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", utils.ErrNotOnvif, resp.Status)
	}

	// Bounded and checked: this body is whatever the device chose to send, and the error
	// used to be discarded, so a truncated read surfaced later as a puzzling etree parse
	// failure or -- worse -- as an endpoint map that silently came back empty.
	doc := etree.NewDocument()
	data, err := io.ReadAll(io.LimitReader(resp.Body, networking.MaxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("GetCapabilities: %w", err)
	}
	if len(data) > networking.MaxResponseBytes {
		return nil, fmt.Errorf("GetCapabilities: reply exceeds %d bytes", networking.MaxResponseBytes)
	}
	if err := doc.ReadFromBytes(data); err != nil {
		return nil, fmt.Errorf("GetCapabilities: %w", err)
	}
	services := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/*/XAddr")
	for _, j := range services {
		dw.client.AddEndpoint(j.Parent().Tag, j.Text())
	}
	extension_services := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/Extension/*/XAddr")
	for _, j := range extension_services {
		dw.client.AddEndpoint(j.Parent().Tag, j.Text())
	}

	return dw, nil
}

func (dw *deviceWrapper) GetUUID() string { return dw.client.GetUUID() }

// GetServices return available endpoints
func (dw *deviceWrapper) GetServices() map[string]string { return dw.client.GetServices() }

// GetEndpoint returns specific ONVIF service endpoint address
func (dw *deviceWrapper) GetEndpoint(name string) string { return dw.client.GetEndpoint(name) }

func (dw *deviceWrapper) GetDeviceEndpoint() string { return dw.GetEndpoint("device") }

// FetchStreamURI returns the stream URI of the appliance's first media profile, or "" when
// it has none.
//
// It carries no credentials, in either direction. It used to interpolate our own username
// and password into the RTSP URL, which put a password into a string callers log, print and
// pass to other processes -- against the rule in AGENTS.md that a secret must not reach a
// dump. It now also drops any account the *device* embedded in its answer, which firmware
// commonly does: rtsp://admin:secret@host/... is an ordinary reply to GetStreamUri, and this
// string is printed on stdout by `onvif-cli streams`.
//
// So a URI from here will not authenticate on its own, and that is the deliberate cost. A
// caller that needs authenticated RTSP adds its own credentials at the point of use, where it
// can decide how they are handled -- which is the only place that decision can be made
// safely, since this package cannot know whether its result is about to be logged.
//
// "First" is now the lowest profile token in lexicographic order. It used to be whichever
// key the map yielded first, so an appliance with several profiles answered differently
// between runs -- the same defect HasEndpoint was fixed for.
//
// And it is the first profile that actually has a URI, not simply the first profile. A
// profile whose media configuration is incomplete -- no video encoder attached -- faults
// GetStreamUri, and FetchMediaProfileUris swallows that by design and leaves the field empty.
// Returning the lowest-sorting slot regardless therefore reported "no stream" for a camera
// that was streaming from its next profile, and Profile_1 before Profile_2 is the common
// vendor naming. Empty is now reserved for what the doc says it means: no profile has one.
func (p *ProfileS) FetchStreamURI(ctx context.Context) string {
	profiles := p.FetchMediaProfiles(ctx)

	for _, token := range slices.Sorted(maps.Keys(profiles.Profiles)) {
		if uri := string(profiles.Profiles[token].Uris.Stream.Uri); uri != "" {
			return uri
		}
	}
	return ""
}

// ProfileS returns the Profile S client for this appliance.
func (dw *deviceWrapper) ProfileS() (*ProfileS, bool) { return NewProfileS(dw.client) }

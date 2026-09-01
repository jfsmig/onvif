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

package sdk

import (
	"context"
	"fmt"
	"github.com/jfsmig/onvif/utils"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/rs/zerolog"

	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/media"
	"github.com/jfsmig/onvif/networking"
)

//go:generate go run github.com/jfsmig/onvif/bin/onvif-codegen profile sdk ./profiles

var (
	// Logger is a zerolog logger, that can be safely used from any part of the application.
	// It gathers the format and the output. The application can replace the default Logger
	// for an alternative that meets its own output.
	Logger = zerolog.
		New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().
		Logger()
)

// Appliance is one connection to one ONVIF device.
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
	Video        Video
	Audio        Audio
	Capabilities media.Capabilities
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
	// CallMethod returns a nil response together with its error, so the body must not be
	// touched before err is checked: an unreachable device used to panic here.
	resp, err := dw.client.CallMethod(ctx, device.GetSystemDateAndTime{})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", utils.ErrNotOnvif, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", utils.ErrNotOnvif, resp.Status)
	}

	resp, err = dw.client.CallMethod(ctx, device.GetCapabilities{Category: "All"})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", utils.ErrNotOnvif, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", utils.ErrNotOnvif, resp.Status)
	}

	doc := etree.NewDocument()
	data, _ := io.ReadAll(resp.Body)

	if err := doc.ReadFromBytes(data); err != nil {
		return nil, err
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

// FetchStreamURI returns a stream URI with the credentials interpolated into it.
//
// Two long-standing defects, kept as they were because fixing them here would hide a
// behaviour change inside an API move: the profile is chosen by map iteration order, so an
// appliance with several profiles yields a different answer between runs; and the password
// is written into a URL the caller is likely to log.
func (p *ProfileS) FetchStreamURI(ctx context.Context) string {
	profiles := p.FetchMediaProfiles(ctx)
	for k := range profiles.Profiles {
		streamURI := string(profiles.Profiles[k].Uris.Stream.Uri)
		auth := p.client.GetAuth()
		return strings.Replace(streamURI, "rtsp://", "rtsp://"+auth.Username+":"+auth.Password+"@", 1)
	}
	return ""
}

// ProfileS returns the Profile S client for this appliance.
func (dw *deviceWrapper) ProfileS() (*ProfileS, bool) { return NewProfileS(dw.client) }

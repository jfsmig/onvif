package sdk

import (
	"context"
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

var (
	// Logger is a zerolog logger, that can be safely used from any part of the application.
	// It gathers the format and the output. The application can replace the default Logger
	// for an alternative that meets its own output.
	Logger = zerolog.
		New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().
		Logger()
)

type Appliance interface {
	// GetUUID return the unique identifier of the remote appliance.
	// The UUID is Usually known after the discovery.
	// TODO(jfs): determine the UUID for devices that do not requires discovery
	GetUUID() string

	GetDeviceEndpoint() string

	GetEndpoint(name string) string

	GetServices() map[string]string

	FetchStreamURI(ctx context.Context) string

	FetchDeviceDescriptor(ctx context.Context) DeviceDescriptor

	// FetchDeviceNetwork fetches from the Appliance the network configuration from the Device
	FetchDeviceNetwork(ctx context.Context) DeviceNetwork

	// FetchDeviceNetwork fetches from the Appliance the security configuration from the Device
	FetchDeviceSecurity(ctx context.Context) DeviceSecurity

	// FetchDeviceNetwork fetches from the Appliance the core system configuration from the Device
	FetchDeviceSystem(ctx context.Context) DeviceSystem

	// FetchMedia fetches from the Appliance a fully-hydrated Media structure
	FetchMedia(ctx context.Context) Media

	// FetchPTZ fetches from the Appliance a fully-hydrated Ptz structure
	FetchPTZ(ctx context.Context) Ptz

	// FetchEvent fetches from the Appliance a fully-hydrated Event structure
	FetchEvent(ctx context.Context) Event

	// FetchMediaProfiles fetches from the Appliance only the Profile related information,
	// otherwise part from the Media information
	FetchProfiles(ctx context.Context) Profiles
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
	resp, err := dw.client.CallMethod(ctx, device.GetSystemDateAndTime{})
	resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, utils.ErrNotOnvif
	}

	resp, err = dw.client.CallMethod(ctx, device.GetCapabilities{Category: "All"})
	defer resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, utils.ErrNotOnvif
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

func (dw *deviceWrapper) FetchStreamURI(ctx context.Context) string {
	profiles := dw.FetchProfiles(ctx)
	for k, _ := range profiles.Profiles {
		streamURI := string(profiles.Profiles[k].Uris.Stream.Uri)
		auth := dw.client.GetAuth()
		return strings.Replace(streamURI, "rtsp://", "rtsp://"+auth.Username+":"+auth.Password+"@", 1)
	}
	return ""
}

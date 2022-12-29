package sdk

import (
	"context"
	"github.com/jfsmig/onvif/errorz"
	"github.com/jfsmig/onvif/networking"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/beevik/etree"
	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/media"

	"github.com/rs/zerolog"
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

	GetEndpoint(name string) string

	GetServices() map[string]string

	FetchDeviceDescriptor(ctx context.Context) DeviceDescriptor

	// FetchMedia fetches from the Appliance a fully-hydrated Media structure
	FetchMedia(ctx context.Context) Media

	// FetchPTZ fetches from the Appliance a fully-hydrated Ptz structure
	FetchPTZ(ctx context.Context) Ptz

	// FetchDevice fetches from the Appliance a fully-hydrated Device structure
	FetchDeviceNetwork(ctx context.Context) DeviceNetwork
	FetchDeviceSecurity(ctx context.Context) DeviceSecurity
	FetchDeviceSystem(ctx context.Context) DeviceSystem

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

func NewDevice(params networking.ClientParams) (Appliance, error) {
	client, err := networking.NewClient(params, "")
	if err != nil {
		return nil, err
	}
	dw := &deviceWrapper{client: client}
	return dw.load()
}

func WrapClient(client *networking.Client) (Appliance, error) {
	dw := &deviceWrapper{client: client}
	return dw.load()
}

func (dw *deviceWrapper) load() (Appliance, error) {
	resp, err := dw.client.CallMethod(device.GetSystemDateAndTime{})
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errorz.ErrNotOnvif
	}

	resp, err = dw.client.CallMethod(device.GetCapabilities{Category: "All"})
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errorz.ErrNotOnvif
	}

	doc := etree.NewDocument()
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

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

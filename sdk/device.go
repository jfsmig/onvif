package sdk

import (
	"context"
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/xsd/onvif"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/beevik/etree"
	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/media"
	"github.com/juju/errors"

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

type Device interface {
	FetchDescriptor(ctx context.Context) DeviceDescriptor

	// FetchMedia fetches from the Device a fully-hydrated Media structure
	FetchMedia(ctx context.Context) Media

	// FetchVideo fetches from the Device only the Video related information,
	// otherwise part from the Media information
	FetchVideo(ctx context.Context) Video

	// FetchAudio fetches from the Device only the Audio related information,
	// otherwise part from the Media information
	FetchAudio(ctx context.Context) Audio

	// FetchAudio fetches from the Device only the Profile related information,
	// otherwise part from the Media information
	FetchProfiles(ctx context.Context) Profiles
}

type DeviceDescriptor struct {
	Capabilities onvif.Capabilities
	Info         device.GetDeviceInformationResponse
}

type Media struct {
	Video        Video
	Audio        Audio
	Profiles     Profiles
	Capabilities media.Capabilities
}

type deviceWrapper struct {
	client *networking.Client
}

func WrapClient(client *networking.Client) (Device, error) {
	dw := &deviceWrapper{client: client}
	return dw.load()
}

// GetServices return available endpoints
func (dw *deviceWrapper) GetServices() map[string]string {
	return dw.client.GetServices()
}

// GetEndpoint returns specific ONVIF service endpoint address
func (dw *deviceWrapper) GetEndpoint(name string) string {
	return dw.client.GetEndpoint(name)
}

func NewDevice(params networking.ClientParams) (Device, error) {
	client, err := networking.NewClient(params)
	if err != nil {
		return nil, err
	}
	dw := &deviceWrapper{client: client}
	return dw.load()
}

func (dw *deviceWrapper) load() (Device, error) {
	resp, err := dw.client.CallMethod(device.GetCapabilities{Category: "All"})
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errors.New("camera not available or ONVIF not supported")
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

func (dw *deviceWrapper) FetchDescriptor(ctx context.Context) DeviceDescriptor {
	out := DeviceDescriptor{}

	if info, err := device.Call_GetDeviceInformation(context.Background(), dw.client, device.GetDeviceInformation{}); err == nil {
		out.Info = info
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetDeviceInformation").Msg("device")
	}

	if caps, err := device.Call_GetCapabilities(ctx, dw.client, device.GetCapabilities{}); err == nil {
		out.Capabilities = caps.Capabilities
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCapabilities").Msg("device")
	}
	return out
}

func (dw *deviceWrapper) FetchMedia(ctx context.Context) Media {
	out := Media{}

	if caps, err := media.Call_GetServiceCapabilities(ctx, dw.client, media.GetServiceCapabilities{}); err == nil {
		out.Capabilities = caps.Capabilities
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("media")
	}

	out.Video = dw.FetchVideo(ctx)
	out.Audio = dw.FetchAudio(ctx)
	out.Profiles = dw.FetchProfiles(ctx)
	return out
}

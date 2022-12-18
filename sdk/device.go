package sdk

import (
	"context"
	"github.com/use-go/onvif/networking"
	"github.com/use-go/onvif/xsd/onvif"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/beevik/etree"
	"github.com/juju/errors"
	"github.com/use-go/onvif/device"
	"github.com/use-go/onvif/media"

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

// Why not
type deviceWrapper struct {
	client *networking.Client
}

func NewDevice(params networking.ClientParams) (Device, error) {
	client, err := networking.NewClient(params)
	if err != nil {
		return nil, err
	}

	dev := &deviceWrapper{client: client}

	resp, err := dev.client.CallMethod(device.GetCapabilities{Category: "All"})
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errors.New("camera not available or ONVIF not supported")
	}

	doc := etree.NewDocument()
	data, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err := doc.ReadFromBytes(data); err != nil {
		return nil, err
	}
	services := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/*/XAddr")
	for _, j := range services {
		dev.client.AddEndpoint(j.Parent().Tag, j.Text())
	}
	extension_services := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/Extension/*/XAddr")
	for _, j := range extension_services {
		dev.client.AddEndpoint(j.Parent().Tag, j.Text())
	}

	return dev, nil
}

func (d *deviceWrapper) FetchDescriptor(ctx context.Context) DeviceDescriptor {
	out := DeviceDescriptor{}

	if info, err := device.Call_GetDeviceInformation(context.Background(), d.client, device.GetDeviceInformation{}); err == nil {
		out.Info = info
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetDeviceInformation").Msg("device")
	}

	if caps, err := device.Call_GetCapabilities(ctx, d.client, device.GetCapabilities{}); err == nil {
		out.Capabilities = caps.Capabilities
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCapabilities").Msg("device")
	}
	return out
}

func (d *deviceWrapper) FetchMedia(ctx context.Context) Media {
	out := Media{}

	if caps, err := media.Call_GetServiceCapabilities(ctx, d.client, media.GetServiceCapabilities{}); err == nil {
		out.Capabilities = caps.Capabilities
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("media")
	}

	out.Video = d.FetchVideo(ctx)
	out.Audio = d.FetchAudio(ctx)
	out.Profiles = d.FetchProfiles(ctx)
	return out
}

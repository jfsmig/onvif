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
	"sync"

	"github.com/jfsmig/onvif/v2/device"
	"github.com/jfsmig/onvif/v2/xsd/onvif"
)

type DeviceDescriptor struct {
	Capabilities        *onvif.Capabilities
	ServiceCapabilities *device.DeviceServiceCapabilities
	Information         *device.GetDeviceInformationResponse
	Service             []device.Service
	SystemDateAndTime   *onvif.SystemDateTime
	Scopes              []onvif.Scope
	EndpointReference   string
	WsdlUrl             string
}

type DeviceSystem struct {
	SystemLog             *string
	AccessLog             *string
	SupportInformation    string
	DiscoveryMode         string
	RemoteDiscoveryMode   string
	Location              *onvif.LocationEntity
	Hostname              *onvif.HostnameInformation
	SystemUris            *device.GetSystemUrisResponse
	StorageConfigurations []device.StorageConfiguration
}

type DeviceSecurity struct {
	RemoteUser   *onvif.RemoteUser
	Users        []onvif.User
	AccessPolicy *onvif.BinaryData
	// A pointer, unlike a plain bool, because every other field of this struct is one or
	// a slice: absence was representable for all of them and not for this. A faulted
	// GetClientCertificateMode dumped as false, which reads as the camera confirming that
	// client certificates are off rather than never having been asked.
	ClientCertificateMode *bool
	NvtCertificate        []CertificateX
	CACertificate         []CertificateX
	CertificateStatus     []onvif.CertificateStatus
}

type DeviceNetwork struct {
	DPAddress          []onvif.NetworkHost
	NetworkProtocols   []onvif.NetworkProtocol
	RelayOutputs       []onvif.RelayOutput
	NICs               map[onvif.ReferenceToken]*NetworkInterfaceX
	Dot1XConfiguration map[onvif.ReferenceToken]*onvif.Dot1XConfiguration
	NTP                *onvif.NTPInformation
	DNS                *onvif.DNSInformation
	DynDNS             *onvif.DynamicDNSInformation
	NetworkGateway     *onvif.NetworkGateway
	ZeroConfiguration  *onvif.NetworkZeroConfiguration
	IPAddressFilter    *onvif.IPAddressFilter
}

// NetworkInterfaceX is one network interface as the device reports it.
//
// It held two more fields, Dot1XConfiguration and Status, and nothing ever assigned either.
// That was worse than inert: `onvif-cli dump device` JSON-encodes this, so every NIC carried
// a populated-looking Dot1XConfiguration with an empty Identity and a Dot11Status with an
// empty SSID -- which an operator reads as the camera reporting no 802.1X identity and no
// wireless association, when neither question was ever asked.
//
// Dot1XConfiguration was also redundant: DeviceNetwork.Dot1XConfiguration already holds every
// configuration the device has, keyed by token, and is actually filled. Status was simply
// never implemented; see the TODO beside FetchDeviceNetwork's other missing call.
type NetworkInterfaceX struct {
	NetworkInterface onvif.NetworkInterface
}

type CertificateX struct {
	Certificate    onvif.Certificate
	Pkcs10Response onvif.BinaryData
	Information    onvif.CertificateInformation
}

func (p *ProfileS) FetchDeviceDescriptor(ctx context.Context) DeviceDescriptor {
	out := DeviceDescriptor{}

	var wg sync.WaitGroup

	wg.Go(func() {
		if capa, err := device.Call_GetCapabilities(ctx, p.client, device.GetCapabilities{Category: "All"}); err == nil {
			// The device's own reply, handed back whole and encoded whole by `dump all`.
			// Its thirteen XAddr fields never pass through networking.AddEndpoint, which
			// only ever sees a copy on the way into the routing table -- so without this a
			// dump showed the account stripped from the GetServices() map and kept in the
			// Capabilities block printed just below it.
			redactURIs(&capa.Capabilities)
			out.Capabilities = &capa.Capabilities
		} else {
			rpcFailure(p.client, err, "GetCapabilities").Msg("device")
		}
	})

	wg.Go(func() {
		if caps, err := device.Call_GetServiceCapabilities(ctx, p.client, device.GetServiceCapabilities{}); err == nil {
			out.ServiceCapabilities = &caps.Capabilities
		} else {
			rpcFailure(p.client, err, "GetServiceCapabilities").Msg("device")
		}
	})

	wg.Go(func() {
		if info, err := device.Call_GetDeviceInformation(ctx, p.client, device.GetDeviceInformation{}); err == nil {
			out.Information = &info
		} else {
			rpcFailure(p.client, err, "GetDeviceInformation").Msg("device")
		}
	})

	wg.Go(func() {
		if srvs, err := device.Call_GetServices(ctx, p.client, device.GetServices{}); err == nil {
			out.Service = srvs.Service
		} else {
			rpcFailure(p.client, err, "GetServices").Msg("device")
		}
	})

	wg.Go(func() {
		if info, err := device.Call_GetSystemDateAndTime(ctx, p.client, device.GetSystemDateAndTime{}); err == nil {
			out.SystemDateAndTime = &info.SystemDateAndTime
		} else {
			rpcFailure(p.client, err, "GetSystemDateAndTime").Msg("device")
		}
	})

	wg.Go(func() {
		if scopes, err := device.Call_GetScopes(ctx, p.client, device.GetScopes{}); err == nil {
			out.Scopes = scopes.Scopes
		} else {
			rpcFailure(p.client, err, "GetScopes").Msg("device")
		}
	})

	wg.Go(func() {
		if url, err := device.Call_GetWsdlUrl(ctx, p.client, device.GetWsdlUrl{}); err == nil {
			out.WsdlUrl = string(url.WsdlUrl)
		} else {
			rpcFailure(p.client, err, "GetWsdlUrl").Msg("device")
		}
	})

	wg.Go(func() {
		if er, err := device.Call_GetEndpointReference(ctx, p.client, device.GetEndpointReference{}); err == nil {
			out.EndpointReference = er.GUID
		} else {
			rpcFailure(p.client, err, "GetEndpointReference").Msg("device")
		}
	})

	wg.Wait()
	return out
}

func (p *ProfileS) FetchDeviceSystem(ctx context.Context) DeviceSystem {
	out := DeviceSystem{}

	var wg sync.WaitGroup

	wg.Go(func() {
		if info, err := device.Call_GetGeoLocation(ctx, p.client, device.GetGeoLocation{}); err == nil {
			out.Location = &info.Location
		} else {
			rpcFailure(p.client, err, "GetGeoLocation").Msg("device")
		}
	})

	wg.Go(func() {
		if log, err := device.Call_GetSystemLog(ctx, p.client, device.GetSystemLog{LogType: "System"}); err == nil {
			out.SystemLog = &log.SystemLog.String
		} else {
			rpcFailure(p.client, err, "GetSystemLog").Str("type", "system").Msg("device")
		}
	})

	wg.Go(func() {
		if log, err := device.Call_GetSystemLog(ctx, p.client, device.GetSystemLog{LogType: "Access"}); err == nil {
			out.AccessLog = &log.SystemLog.String
		} else {
			rpcFailure(p.client, err, "GetSystemLog").Str("type", "access").Msg("device")
		}
	})

	wg.Go(func() {
		if dm, err := device.Call_GetDiscoveryMode(ctx, p.client, device.GetDiscoveryMode{}); err == nil {
			out.DiscoveryMode = string(dm.DiscoveryMode)
		} else {
			rpcFailure(p.client, err, "GetDiscoveryMode").Msg("device")
		}
	})

	wg.Go(func() {
		if dm, err := device.Call_GetRemoteDiscoveryMode(ctx, p.client, device.GetRemoteDiscoveryMode{}); err == nil {
			out.RemoteDiscoveryMode = string(dm.RemoteDiscoveryMode)
		} else {
			rpcFailure(p.client, err, "GetRemoteDiscoveryMode").Msg("device")
		}
	})

	wg.Go(func() {
		if hi, err := device.Call_GetHostname(ctx, p.client, device.GetHostname{}); err == nil {
			out.Hostname = &hi.HostnameInformation
		} else {
			rpcFailure(p.client, err, "GetHostname").Msg("device")
		}
	})

	wg.Go(func() {
		if uris, err := device.Call_GetSystemUris(ctx, p.client, device.GetSystemUris{}); err == nil {
			out.SystemUris = &uris
		} else {
			rpcFailure(p.client, err, "GetSystemUris").Msg("device")
		}
	})

	wg.Go(func() {
		if si, err := device.Call_GetSystemSupportInformation(ctx, p.client, device.GetSystemSupportInformation{}); err == nil {
			out.SupportInformation = si.SupportInformation.String
		} else {
			rpcFailure(p.client, err, "GetSystemSupportInformation").Msg("device")
		}
	})

	wg.Go(func() {
		if configs, err := device.Call_GetStorageConfigurations(ctx, p.client, device.GetStorageConfigurations{}); err == nil {
			out.StorageConfigurations = configs.StorageConfigurations
		} else {
			rpcFailure(p.client, err, "GetStorageConfigurations").Msg("device")
		}
	})

	wg.Wait()
	return out
}

func (p *ProfileS) FetchDeviceSecurity(ctx context.Context) DeviceSecurity {
	out := DeviceSecurity{}

	var wg sync.WaitGroup

	wg.Go(func() {
		if ru, err := device.Call_GetRemoteUser(ctx, p.client, device.GetRemoteUser{}); err == nil {
			out.RemoteUser = &ru.RemoteUser
		} else {
			rpcFailure(p.client, err, "GetRemoteUser").Msg("device")
		}
	})

	wg.Go(func() {
		if users, err := device.Call_GetUsers(ctx, p.client, device.GetUsers{}); err == nil {
			out.Users = users.User
		} else {
			rpcFailure(p.client, err, "GetUsers").Msg("device")
		}
	})

	wg.Go(func() {
		if ap, err := device.Call_GetAccessPolicy(ctx, p.client, device.GetAccessPolicy{}); err == nil {
			out.AccessPolicy = &ap.PolicyFile
		} else {
			rpcFailure(p.client, err, "GetAccessPolicy").Msg("device")
		}
	})

	wg.Go(func() {
		if certs, err := device.Call_GetCertificates(ctx, p.client, device.GetCertificates{}); err == nil {
			for _, cert := range certs.NvtCertificate {
				latest := CertificateX{Certificate: cert}
				p.loadCertificate(ctx, &latest)
				out.NvtCertificate = append(out.NvtCertificate, latest)
			}
		} else {
			rpcFailure(p.client, err, "GetCertificates").Msg("device")
		}
	})

	wg.Go(func() {
		if cs, err := device.Call_GetCertificatesStatus(ctx, p.client, device.GetCertificatesStatus{}); err == nil {
			out.CertificateStatus = cs.CertificateStatus
		} else {
			rpcFailure(p.client, err, "GetCertificatesStatus").Msg("device")
		}
	})

	wg.Go(func() {
		if cs, err := device.Call_GetClientCertificateMode(ctx, p.client, device.GetClientCertificateMode{}); err == nil {
			enabled := bool(cs.Enabled)
			out.ClientCertificateMode = &enabled
		} else {
			rpcFailure(p.client, err, "GetClientCertificateMode").Msg("device")
		}
	})

	wg.Go(func() {
		if cs, err := device.Call_GetCACertificates(ctx, p.client, device.GetCACertificates{}); err == nil {
			for _, cert := range cs.CACertificate {
				latest := CertificateX{Certificate: cert}
				p.loadCertificate(ctx, &latest)
				out.CACertificate = append(out.CACertificate, latest)
			}
		} else {
			rpcFailure(p.client, err, "GetCACertificates").Msg("device")
		}
	})

	wg.Wait()
	return out
}

func (p *ProfileS) FetchDeviceNetwork(ctx context.Context) DeviceNetwork {
	out := DeviceNetwork{
		NICs:               make(map[onvif.ReferenceToken]*NetworkInterfaceX),
		Dot1XConfiguration: make(map[onvif.ReferenceToken]*onvif.Dot1XConfiguration),
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		if dpa, err := device.Call_GetDPAddresses(ctx, p.client, device.GetDPAddresses{}); err == nil {
			out.DPAddress = dpa.DPAddress
		} else {
			rpcFailure(p.client, err, "GetDPAddresses").Msg("device")
		}
	})

	wg.Go(func() {
		if dns, err := device.Call_GetDNS(ctx, p.client, device.GetDNS{}); err == nil {
			out.DNS = &dns.DNSInformation
		} else {
			rpcFailure(p.client, err, "GetDNS").Msg("device")
		}
	})

	wg.Go(func() {
		if ddns, err := device.Call_GetDynamicDNS(ctx, p.client, device.GetDynamicDNS{}); err == nil {
			out.DynDNS = &ddns.DynamicDNSInformation
		} else {
			rpcFailure(p.client, err, "GetDynamicDNS").Msg("device")
		}
	})

	wg.Go(func() {
		if ntp, err := device.Call_GetNTP(ctx, p.client, device.GetNTP{}); err == nil {
			out.NTP = &ntp.NTPInformation
		} else {
			rpcFailure(p.client, err, "GetNTP").Msg("device")
		}
	})

	wg.Go(func() {
		if nics, err := device.Call_GetNetworkInterfaces(ctx, p.client, device.GetNetworkInterfaces{}); err == nil {
			for _, nic := range nics.NetworkInterfaces {
				latest := &NetworkInterfaceX{NetworkInterface: nic}

				out.NICs[latest.NetworkInterface.Token] = latest
			}
		} else {
			rpcFailure(p.client, err, "GetNetworkInterfaces").Msg("device")
		}
	})

	wg.Go(func() {
		if protos, err := device.Call_GetNetworkProtocols(ctx, p.client, device.GetNetworkProtocols{}); err == nil {
			out.NetworkProtocols = protos.NetworkProtocols
		} else {
			rpcFailure(p.client, err, "GetNetworkProtocols").Msg("device")
		}
	})

	wg.Go(func() {
		if dgw, err := device.Call_GetNetworkDefaultGateway(ctx, p.client, device.GetNetworkDefaultGateway{}); err == nil {
			out.NetworkGateway = &dgw.NetworkGateway
		} else {
			rpcFailure(p.client, err, "GetNetworkDefaultGateway").Msg("device")
		}
	})

	wg.Go(func() {
		if zc, err := device.Call_GetZeroConfiguration(ctx, p.client, device.GetZeroConfiguration{}); err == nil {
			out.ZeroConfiguration = &zc.ZeroConfiguration
		} else {
			rpcFailure(p.client, err, "GetZeroConfiguration").Msg("device")
		}
	})

	wg.Go(func() {
		if iaf, err := device.Call_GetIPAddressFilter(ctx, p.client, device.GetIPAddressFilter{}); err == nil {
			out.IPAddressFilter = &iaf.IPAddressFilter
		} else {
			rpcFailure(p.client, err, "GetIPAddressFilter").Msg("device")
		}
	})

	wg.Go(func() {
		if cs, err := device.Call_GetRelayOutputs(ctx, p.client, device.GetRelayOutputs{}); err == nil {
			out.RelayOutputs = cs.RelayOutputs
		} else {
			rpcFailure(p.client, err, "GetRelayOutputs").Msg("device")
		}
	})

	wg.Go(func() {
		if x, err := device.Call_GetDot1XConfigurations(ctx, p.client, device.GetDot1XConfigurations{}); err == nil {
			for _, cfg := range x.Dot1XConfiguration {
				cfgCopy := cfg
				out.Dot1XConfiguration[cfg.Dot1XConfigurationToken] = &cfgCopy
			}
		} else {
			rpcFailure(p.client, err, "GetDot1XConfigurations").Msg("device")
		}
	})

	// TODO(jfsmig): ScanAvailableDot11Networks
	// TODO(jfsmig): GetDot11Status per interface, into NetworkInterfaceX. It takes an
	// InterfaceToken, so it is a per-NIC fan-out in the shape of FetchPTZ's inner one.

	wg.Wait()
	return out
}

func (p *ProfileS) loadCertificate(ctx context.Context, out *CertificateX) {
	if cs, err := device.Call_GetPkcs10Request(ctx, p.client, device.GetPkcs10Request{CertificateID: out.Certificate.CertificateID}); err == nil {
		out.Pkcs10Response = cs.Pkcs10Request
	} else {
		rpcFailure(p.client, err, "GetPkcs10Request").Msg("device")
	}

	if cs, err := device.Call_GetCertificateInformation(ctx, p.client, device.GetCertificateInformation{CertificateID: out.Certificate.CertificateID}); err == nil {
		out.Information = cs.CertificateInformation
	} else {
		rpcFailure(p.client, err, "GetCertificateInformation").Msg("device")
	}
}

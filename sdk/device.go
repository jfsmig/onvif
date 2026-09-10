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

	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/xsd/onvif"
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
	RemoteUser            *onvif.RemoteUser
	Users                 []onvif.User
	AccessPolicy          *onvif.BinaryData
	ClientCertificateMode bool
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

type NetworkInterfaceX struct {
	NetworkInterface   onvif.NetworkInterface
	Dot1XConfiguration onvif.Dot1XConfiguration
	Status             onvif.Dot11Status
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
			out.Capabilities = &capa.Capabilities
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCapabilities").Msg("device")
		}
	})

	wg.Go(func() {
		if caps, err := device.Call_GetServiceCapabilities(ctx, p.client, device.GetServiceCapabilities{}); err == nil {
			out.ServiceCapabilities = &caps.Capabilities
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("device")
		}
	})

	wg.Go(func() {
		if info, err := device.Call_GetDeviceInformation(ctx, p.client, device.GetDeviceInformation{}); err == nil {
			out.Information = &info
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetDeviceInformation").Msg("device")
		}
	})

	wg.Go(func() {
		if srvs, err := device.Call_GetServices(ctx, p.client, device.GetServices{}); err == nil {
			out.Service = srvs.Service
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetServices").Msg("device")
		}
	})

	wg.Go(func() {
		if info, err := device.Call_GetSystemDateAndTime(ctx, p.client, device.GetSystemDateAndTime{}); err == nil {
			out.SystemDateAndTime = &info.SystemDateAndTime
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetSystemDateAndTime").Msg("device")
		}
	})

	wg.Go(func() {
		if scopes, err := device.Call_GetScopes(ctx, p.client, device.GetScopes{}); err == nil {
			out.Scopes = scopes.Scopes
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetScopes").Msg("device")
		}
	})

	wg.Go(func() {
		if url, err := device.Call_GetWsdlUrl(ctx, p.client, device.GetWsdlUrl{}); err == nil {
			out.WsdlUrl = string(url.WsdlUrl)
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetWsdlUrl").Msg("device")
		}
	})

	wg.Go(func() {
		if er, err := device.Call_GetEndpointReference(ctx, p.client, device.GetEndpointReference{}); err == nil {
			out.EndpointReference = er.GUID
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetEndpointReference").Msg("device")
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
			Logger.Trace().Err(err).Str("rpc", "GetGeoLocation").Msg("device")
		}
	})

	wg.Go(func() {
		if log, err := device.Call_GetSystemLog(ctx, p.client, device.GetSystemLog{LogType: "System"}); err == nil {
			out.SystemLog = &log.SystemLog.String
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetSystemLog").Str("type", "system").Msg("device")
		}
	})

	wg.Go(func() {
		if log, err := device.Call_GetSystemLog(ctx, p.client, device.GetSystemLog{LogType: "Access"}); err == nil {
			out.AccessLog = &log.SystemLog.String
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetSystemLog").Str("type", "access").Msg("device")
		}
	})

	wg.Go(func() {
		if dm, err := device.Call_GetDiscoveryMode(ctx, p.client, device.GetDiscoveryMode{}); err == nil {
			out.DiscoveryMode = string(dm.DiscoveryMode)
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetDiscoveryMode").Msg("device")
		}
	})

	wg.Go(func() {
		if dm, err := device.Call_GetRemoteDiscoveryMode(ctx, p.client, device.GetRemoteDiscoveryMode{}); err == nil {
			out.RemoteDiscoveryMode = string(dm.RemoteDiscoveryMode)
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetRemoteDiscoveryMode").Msg("device")
		}
	})

	wg.Go(func() {
		if hi, err := device.Call_GetHostname(ctx, p.client, device.GetHostname{}); err == nil {
			out.Hostname = &hi.HostnameInformation
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetHostname").Msg("device")
		}
	})

	wg.Go(func() {
		if uris, err := device.Call_GetSystemUris(ctx, p.client, device.GetSystemUris{}); err == nil {
			out.SystemUris = &uris
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetSystemUris").Msg("device")
		}
	})

	wg.Go(func() {
		if si, err := device.Call_GetSystemSupportInformation(ctx, p.client, device.GetSystemSupportInformation{}); err == nil {
			out.SupportInformation = si.SupportInformation.String
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetSystemSupportInformation").Msg("device")
		}
	})

	wg.Go(func() {
		if configs, err := device.Call_GetStorageConfigurations(ctx, p.client, device.GetStorageConfigurations{}); err == nil {
			out.StorageConfigurations = configs.StorageConfigurations
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetStorageConfigurations").Msg("device")
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
			Logger.Trace().Err(err).Str("rpc", "GetRemoteUser").Msg("device")
		}
	})

	wg.Go(func() {
		if users, err := device.Call_GetUsers(ctx, p.client, device.GetUsers{}); err == nil {
			out.Users = users.User
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetUsers").Msg("device")
		}
	})

	wg.Go(func() {
		if ap, err := device.Call_GetAccessPolicy(ctx, p.client, device.GetAccessPolicy{}); err == nil {
			out.AccessPolicy = &ap.PolicyFile
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetAccessPolicy").Msg("device")
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
			Logger.Trace().Err(err).Str("rpc", "GetCertificates").Msg("device")
		}
	})

	wg.Go(func() {
		if cs, err := device.Call_GetCertificatesStatus(ctx, p.client, device.GetCertificatesStatus{}); err == nil {
			out.CertificateStatus = cs.CertificateStatus
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetCertificatesStatus").Msg("device")
		}
	})

	wg.Go(func() {
		if cs, err := device.Call_GetClientCertificateMode(ctx, p.client, device.GetClientCertificateMode{}); err == nil {
			out.ClientCertificateMode = bool(cs.Enabled)
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetClientCertificateMode").Msg("device")
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
			Logger.Trace().Err(err).Str("rpc", "GetCACertificates").Msg("device")
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
			Logger.Trace().Err(err).Str("rpc", "GetDPAddresses").Msg("device")
		}
	})

	wg.Go(func() {
		if dns, err := device.Call_GetDNS(ctx, p.client, device.GetDNS{}); err == nil {
			out.DNS = &dns.DNSInformation
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetDNS").Msg("device")
		}
	})

	wg.Go(func() {
		if ddns, err := device.Call_GetDynamicDNS(ctx, p.client, device.GetDynamicDNS{}); err == nil {
			out.DynDNS = &ddns.DynamicDNSInformation
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetDynamicDNS").Msg("device")
		}
	})

	wg.Go(func() {
		if ntp, err := device.Call_GetNTP(ctx, p.client, device.GetNTP{}); err == nil {
			out.NTP = &ntp.NTPInformation
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetNTP").Msg("device")
		}
	})

	wg.Go(func() {
		if nics, err := device.Call_GetNetworkInterfaces(ctx, p.client, device.GetNetworkInterfaces{}); err == nil {
			for _, nic := range nics.NetworkInterfaces {
				latest := &NetworkInterfaceX{NetworkInterface: nic}

				out.NICs[latest.NetworkInterface.Token] = latest
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetNetworkInterfaces").Msg("device")
		}
	})

	wg.Go(func() {
		if protos, err := device.Call_GetNetworkProtocols(ctx, p.client, device.GetNetworkProtocols{}); err == nil {
			out.NetworkProtocols = protos.NetworkProtocols
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetNetworkProtocols").Msg("device")
		}
	})

	wg.Go(func() {
		if dgw, err := device.Call_GetNetworkDefaultGateway(ctx, p.client, device.GetNetworkDefaultGateway{}); err == nil {
			out.NetworkGateway = &dgw.NetworkGateway
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetNetworkDefaultGateway").Msg("device")
		}
	})

	wg.Go(func() {
		if zc, err := device.Call_GetZeroConfiguration(ctx, p.client, device.GetZeroConfiguration{}); err == nil {
			out.ZeroConfiguration = &zc.ZeroConfiguration
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetZeroConfiguration").Msg("device")
		}
	})

	wg.Go(func() {
		if iaf, err := device.Call_GetIPAddressFilter(ctx, p.client, device.GetIPAddressFilter{}); err == nil {
			out.IPAddressFilter = &iaf.IPAddressFilter
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetIPAddressFilter").Msg("device")
		}
	})

	wg.Go(func() {
		if cs, err := device.Call_GetRelayOutputs(ctx, p.client, device.GetRelayOutputs{}); err == nil {
			out.RelayOutputs = cs.RelayOutputs
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetRelayOutputs").Msg("device")
		}
	})

	wg.Go(func() {
		if x, err := device.Call_GetDot1XConfigurations(ctx, p.client, device.GetDot1XConfigurations{}); err == nil {
			for _, cfg := range x.Dot1XConfiguration {
				cfgCopy := cfg
				out.Dot1XConfiguration[cfg.Dot1XConfigurationToken] = &cfgCopy
			}
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetDot1XConfigurations").Msg("device")
		}
	})

	// TODO(jfsmig): ScanAvailableDot11Networks

	wg.Wait()
	return out
}

func (p *ProfileS) loadCertificate(ctx context.Context, out *CertificateX) {
	if cs, err := device.Call_GetPkcs10Request(ctx, p.client, device.GetPkcs10Request{CertificateID: out.Certificate.CertificateID}); err == nil {
		out.Pkcs10Response = cs.Pkcs10Request
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetPkcs10Request").Msg("device")
	}

	if cs, err := device.Call_GetCertificateInformation(ctx, p.client, device.GetCertificateInformation{CertificateID: out.Certificate.CertificateID}); err == nil {
		out.Information = cs.CertificateInformation
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCertificateInformation").Msg("device")
	}
}

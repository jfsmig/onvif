package sdk

import (
	"context"

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

func (dw *deviceWrapper) FetchDeviceDescriptor(ctx context.Context) DeviceDescriptor {
	out := DeviceDescriptor{}

	if capa, err := device.Call_GetCapabilities(ctx, dw.client, device.GetCapabilities{Category: "All"}); err == nil {
		out.Capabilities = &capa.Capabilities
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCapabilities").Msg("device")
	}

	if caps, err := device.Call_GetServiceCapabilities(ctx, dw.client, device.GetServiceCapabilities{}); err == nil {
		out.ServiceCapabilities = &caps.Capabilities
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("device")
	}

	if info, err := device.Call_GetDeviceInformation(ctx, dw.client, device.GetDeviceInformation{}); err == nil {
		out.Information = &info
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("device")
	}

	if srvs, err := device.Call_GetServices(ctx, dw.client, device.GetServices{}); err == nil {
		out.Service = srvs.Service
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("device")
	}

	if info, err := device.Call_GetSystemDateAndTime(ctx, dw.client, device.GetSystemDateAndTime{}); err == nil {
		out.SystemDateAndTime = &info.SystemDateAndTime
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("device")
	}

	if scopes, err := device.Call_GetScopes(ctx, dw.client, device.GetScopes{}); err == nil {
		out.Scopes = scopes.Scopes
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetScopes").Msg("device")
	}

	if url, err := device.Call_GetWsdlUrl(ctx, dw.client, device.GetWsdlUrl{}); err == nil {
		out.WsdlUrl = string(url.WsdlUrl)
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetWsdlUrl").Msg("device")
	}

	if er, err := device.Call_GetEndpointReference(ctx, dw.client, device.GetEndpointReference{}); err == nil {
		out.EndpointReference = er.GUID
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetEndpointReference").Msg("device")
	}

	return out
}

func (dw *deviceWrapper) FetchDeviceSystem(ctx context.Context) DeviceSystem {
	out := DeviceSystem{}

	if info, err := device.Call_GetGeoLocation(ctx, dw.client, device.GetGeoLocation{}); err == nil {
		out.Location = &info.Location
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetGeoLocation").Msg("device")
	}

	if log, err := device.Call_GetSystemLog(ctx, dw.client, device.GetSystemLog{LogType: "System"}); err == nil {
		out.SystemLog = &log.SystemLog.String
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetSystemLog").Str("type", "system").Msg("device")
	}

	if log, err := device.Call_GetSystemLog(ctx, dw.client, device.GetSystemLog{LogType: "Access"}); err == nil {
		out.AccessLog = &log.SystemLog.String
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetSystemLog").Str("type", "access").Msg("device")
	}

	if dm, err := device.Call_GetDiscoveryMode(ctx, dw.client, device.GetDiscoveryMode{}); err == nil {
		out.DiscoveryMode = string(dm.DiscoveryMode)
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetDiscoveryMode").Msg("device")
	}

	if dm, err := device.Call_GetRemoteDiscoveryMode(ctx, dw.client, device.GetRemoteDiscoveryMode{}); err == nil {
		out.RemoteDiscoveryMode = string(dm.RemoteDiscoveryMode)
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetRemoteDiscoveryMode").Msg("device")
	}

	if hi, err := device.Call_GetHostname(ctx, dw.client, device.GetHostname{}); err == nil {
		out.Hostname = &hi.HostnameInformation
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetHostname").Msg("device")
	}

	if uris, err := device.Call_GetSystemUris(ctx, dw.client, device.GetSystemUris{}); err == nil {
		out.SystemUris = &uris
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetSystemUris").Msg("device")
	}

	if si, err := device.Call_GetSystemSupportInformation(ctx, dw.client, device.GetSystemSupportInformation{}); err == nil {
		out.SupportInformation = si.SupportInformation.String
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetSystemSupportInformation").Msg("device")
	}

	if configs, err := device.Call_GetStorageConfigurations(ctx, dw.client, device.GetStorageConfigurations{}); err == nil {
		out.StorageConfigurations = configs.StorageConfigurations
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetStorageConfigurations").Msg("device")
	}

	return out
}

func (dw *deviceWrapper) FetchDeviceSecurity(ctx context.Context) DeviceSecurity {
	out := DeviceSecurity{}

	if ru, err := device.Call_GetRemoteUser(ctx, dw.client, device.GetRemoteUser{}); err == nil {
		out.RemoteUser = &ru.RemoteUser
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetRemoteUser").Msg("device")
	}

	if users, err := device.Call_GetUsers(ctx, dw.client, device.GetUsers{}); err == nil {
		out.Users = users.User
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetUsers").Msg("device")
	}

	if ap, err := device.Call_GetAccessPolicy(ctx, dw.client, device.GetAccessPolicy{}); err == nil {
		out.AccessPolicy = &ap.PolicyFile
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetAccessPolicy").Msg("device")
	}

	if certs, err := device.Call_GetCertificates(ctx, dw.client, device.GetCertificates{}); err == nil {
		for _, cert := range certs.NvtCertificate {
			latest := CertificateX{Certificate: cert}
			dw.loadCertificate(ctx, &latest)
			out.NvtCertificate = append(out.NvtCertificate, latest)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCertificates").Msg("device")
	}

	if cs, err := device.Call_GetCertificatesStatus(ctx, dw.client, device.GetCertificatesStatus{}); err == nil {
		out.CertificateStatus = cs.CertificateStatus
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCertificatesStatus").Msg("device")
	}

	if cs, err := device.Call_GetClientCertificateMode(ctx, dw.client, device.GetClientCertificateMode{}); err == nil {
		out.ClientCertificateMode = bool(cs.Enabled)
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetClientCertificateMode").Msg("device")
	}

	if cs, err := device.Call_GetCACertificates(ctx, dw.client, device.GetCACertificates{}); err == nil {
		for _, cert := range cs.CACertificate {
			latest := CertificateX{Certificate: cert}
			dw.loadCertificate(ctx, &latest)
			out.CACertificate = append(out.CACertificate, latest)
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetCACertificates").Msg("device")
	}

	return out
}

func (dw *deviceWrapper) FetchDeviceNetwork(ctx context.Context) DeviceNetwork {
	out := DeviceNetwork{
		NICs:               make(map[onvif.ReferenceToken]*NetworkInterfaceX),
		Dot1XConfiguration: make(map[onvif.ReferenceToken]*onvif.Dot1XConfiguration),
	}

	if dpa, err := device.Call_GetDPAddresses(ctx, dw.client, device.GetDPAddresses{}); err == nil {
		out.DPAddress = dpa.DPAddress
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetDPAddresses").Msg("device")
	}

	if dns, err := device.Call_GetDNS(ctx, dw.client, device.GetDNS{}); err == nil {
		out.DNS = &dns.DNSInformation
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetDNS").Msg("device")
	}

	if ddns, err := device.Call_GetDynamicDNS(ctx, dw.client, device.GetDynamicDNS{}); err == nil {
		out.DynDNS = &ddns.DynamicDNSInformation
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetDynamicDNS").Msg("device")
	}

	if ntp, err := device.Call_GetNTP(ctx, dw.client, device.GetNTP{}); err == nil {
		out.NTP = &ntp.NTPInformation
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetNTP").Msg("device")
	}

	if nics, err := device.Call_GetNetworkInterfaces(ctx, dw.client, device.GetNetworkInterfaces{}); err == nil {
		for _, nic := range nics.NetworkInterfaces {
			latest := &NetworkInterfaceX{NetworkInterface: nic}

			out.NICs[latest.NetworkInterface.Token] = latest
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetNetworkInterfaces").Msg("device")
	}

	if protos, err := device.Call_GetNetworkProtocols(ctx, dw.client, device.GetNetworkProtocols{}); err == nil {
		out.NetworkProtocols = protos.NetworkProtocols
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetNetworkProtocols").Msg("device")
	}

	if dgw, err := device.Call_GetNetworkDefaultGateway(ctx, dw.client, device.GetNetworkDefaultGateway{}); err == nil {
		out.NetworkGateway = &dgw.NetworkGateway
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetNetworkDefaultGateway").Msg("device")
	}

	if zc, err := device.Call_GetZeroConfiguration(ctx, dw.client, device.GetZeroConfiguration{}); err == nil {
		out.ZeroConfiguration = &zc.ZeroConfiguration
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetZeroConfiguration").Msg("device")
	}

	if iaf, err := device.Call_GetIPAddressFilter(ctx, dw.client, device.GetIPAddressFilter{}); err == nil {
		out.IPAddressFilter = &iaf.IPAddressFilter
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetIPAddressFilter").Msg("device")
	}

	if cs, err := device.Call_GetRelayOutputs(ctx, dw.client, device.GetRelayOutputs{}); err == nil {
		out.RelayOutputs = cs.RelayOutputs
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetRelayOutputs").Msg("device")
	}

	if x, err := device.Call_GetDot1XConfigurations(ctx, dw.client, device.GetDot1XConfigurations{}); err == nil {
		for _, cfg := range x.Dot1XConfiguration {
			cfgCopy := cfg
			out.Dot1XConfiguration[cfg.Dot1XConfigurationToken] = &cfgCopy
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetDot1XConfigurations").Msg("device")
	}

	// TODO(jfsmig): ScanAvailableDot11Networks

	return out
}

func (dw *deviceWrapper) loadCertificate(ctx context.Context, out *CertificateX) {
	if cs, err := device.Call_GetPkcs10Request(ctx, dw.client, device.GetPkcs10Request{CertificateID: out.Certificate.CertificateID}); err == nil {
		out.Pkcs10Response = cs.Pkcs10Request
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetPkcs10Request").Msg("device")
	}

	if cs, err := device.Call_GetCertificateInformation(ctx, dw.client, device.GetCertificateInformation{CertificateID: out.Certificate.CertificateID}); err == nil {
		out.Information = cs.CertificateInformation
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetPkcs10Request").Msg("device")
	}
}

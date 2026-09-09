# Go OnVif Client

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/jfsmig/onvif/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/jfsmig/onvif/tree/master)
[![CodeQL](https://github.com/jfsmig/onvif/actions/workflows/codeql.yml/badge.svg)](https://github.com/jfsmig/onvif/actions/workflows/codeql.yml)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

Simple management IP-devices cameras that honor the [ONVIF Protocol](https://www.onvif.org/) protocol.

The present repository is a fork of [goonvif](https://github.com/use-go/goonvif) that quickly evolved. 
Because of the need for quickly merged changes, the link to the upstream has been cut.

## CLI tools

For the convenience and testing purposes, a [CLI](https://en.wikipedia.org/wiki/Command-line_interface) tool ships
with the repository to help discovering and fetching information from devices.

```console
onvif-cli COMMAND [SUBCOMMAND] [OPTIONS] [ARGUMENTS...]

COMMAND:
  discover [-a]            probe the local networks with WS-Discovery and print one
                           whitespace-separated line per device that answered:
                             INTERFACE XADDR UUID
                           XADDR is the IP:PORT to pass to `dump`; UUID is "-" when the
                           device reported none
  streams [-a]             the same probe, then one line per media profile of every
                           device found:
                             INTERFACE XADDR UUID PROFILE STREAM_URI SNAPSHOT_URI
  dump SUBCOMMAND IP:PORT  print a single JSON object holding a configuration dump of
                           the given camera

  SUBCOMMAND:
    all         the full dump: Descriptor, DeviceSystem, DeviceSecurity, DeviceNetwork,
                Media, Ptz, Profiles, Events
    descriptor  the service endpoints, the UUID and the device descriptor only
    device      the core Device service: Descriptor, DeviceSystem, DeviceSecurity,
                DeviceNetwork
    media       the Media service
    ptz         the PTZ service
    event       the Events service
    profile     the media profiles

OPTIONS (they follow the command — `onvif-cli discover -a`, never precede it):
  -a, --all       for `discover` and `streams`: probe every interface that is up and not
                  loopback. By default a candidate must also be multicast-capable —
                  WS-Discovery is multicast-only — and must not be one of the well-known
                  virtual devices (docker0, br-<id>, veth*, virbr*, cali*, flannel.*,
                  ...). An interface that is down or loopback is never probed either way.
                  A container's eth0 is not in that table, so probing from inside a
                  container needs no option.

ENVIRONMENT:
  ONVIF_USERNAME  credential sent to every camera, "admin" when unset
  ONVIF_PASSWORD  credential sent to every camera, "admin" when unset
```

Data goes to stdout and diagnostics to stderr, so `onvif-cli dump all IP:PORT | jq` works
as it looks. Most commands accept aliases — `find` for `discover`, `events` for `event`,
`prof` for `profile` — which `onvif-cli COMMAND --help` lists.

The default probe set matters on a host running containers, where the virtual interfaces
outnumber the real one by an order of magnitude. Each of them cost a socket, a multicast
join and a line of diagnostics, and none of them can reach a camera.

It also matters for `streams`, and that is worth stating plainly. WS-Discovery replies are
unauthenticated, and `streams` sends `ONVIF_USERNAME` / `ONVIF_PASSWORD` to whatever
answered, over plain HTTP. On the container and VM links the peer that answers is a
container or a guest rather than a camera, so `streams -a` on such a host lets anything
running there collect those credentials. Prefer `-a` on `discover`, which authenticates
nothing.

## SDK

A High Level go package aims at fetching information from the devices:
- [github.com/jfsmig/onvif/sdk](https://pkg.go.dev/github.com/jfsmig/onvif/sdk)

Low-Level go packages implement the OnVIF unitary SOAP calls. For each call :
- [github.com/jfsmig/onvif/device](https://pkg.go.dev/github.com/jfsmig/onvif/device)
- [github.com/jfsmig/onvif/event](https://pkg.go.dev/github.com/jfsmig/onvif/event)
- [github.com/jfsmig/onvif/ptz](https://pkg.go.dev/github.com/jfsmig/onvif/ptz)
- [github.com/jfsmig/onvif/Imaging](https://pkg.go.dev/github.com/jfsmig/onvif/Imaging)
- [github.com/jfsmig/onvif/media](https://pkg.go.dev/github.com/jfsmig/onvif/media)

Helpers:
- [github.com/jfsmig/onvif/networking](https://pkg.go.dev/github.com/jfsmig/onvif/networking)
  implements the low-level SOAP connectivity
- [github.com/jfsmig/go-wsd/wsd](https://pkg.go.dev/github.com/jfsmig/go-wsd/wsd)
  implements the probing of the LAN network interfaces. Please refer to the CLI tool `onvif-cli discover`

### Beginner's Guide

```go
info := networking.ClientInfo{Xaddr: "192.168.1.70:8000"}
auth := networking.ClientAuth{
    Username: os.Getenv("ONVIF_USERNAME"),
    Password: os.Getenv("ONVIF_PASSWORD"),
}

appliance, err := sdk.NewDevice(ctx, info, auth, nil)
if err != nil { /* not a reachable ONVIF device */ }

// The operations are grouped the way the norm groups them: by Profile.
profileS, ok := appliance.ProfileS()
if !ok { /* the appliance advertises no Profile S service */ }

reply, err := profileS.GetProfiles(ctx, media.GetProfiles{})

// PTZ is conditional in Profile S, so ask before assuming.
if profileS.HasPTZ() {
    _, err = profileS.ContinuousMove(ctx, ptz.ContinuousMove{ProfileToken: token})
}
```

### Auto-generated code instead of generics

The low level packages provide one function per OnVIF SOAP method.
Their purpose is to ease the persing and unpacking of the replies.
The problem was the requirement to name the reply field as the reply expected reply type.
But Golang's generics do not provide any sophisticated way to generate the name of a type instead of a type, as the
`#` modifier does with `cpp`. That's why they have all been generated instead of replying on templated functions.

## References

* [OnVif](https://onvif.org)
* [OnVif Specs](https://github.com/onvif/specs)
* [OnVif Discussions](https://github.com/onvif/specs/discussions)

## License

AGPL-3.0-or-later, see [LICENSE](LICENSE). This project began as `goonvif`, later
`use-go/onvif`, distributed under the MIT License; that notice is retained in
[LICENSE.MIT](LICENSE.MIT) and the files derived from it carry both notices in their
headers. See the git history for the full list of contributors.

`docs/` holds ONVIF's own specification and WSDL files, which remain under ONVIF's terms
and are not covered by the AGPL.

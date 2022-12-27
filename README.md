# Go OnVif Client

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CircleCI](https://dl.circleci.com/status-badge/img/gh/jfsmig/onvif/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/jfsmig/onvif/tree/master)

Simple management IP-devices cameras that honor the [ONVIF Protocol](https://www.onvif.org/) protocol.

The present repository is a fork of [goonvif](https://github.com/use-go/goonvif) that quickly evolved. 
Because of the need for quickly merged changes, the link to the upstream has been cut.

## CLI tools

For the convenience and testing purposes, a [CLI](https://en.wikipedia.org/wiki/Command-line_interface) tool ships
with the repository to help discovering and fetching information from devices.

```console
onvif-cli COMMAND [SUBCOMAND] [ARGUMENTS...]
COMMAND:
  discover                 perform a web-service discovery on the local networks and
                           reports one line (IP:PORT CRLF) per device that responded
  dump SUBCOMMAND IP:PORT  Prints a single JSON object with a configuration dump
                           for the given camera
  SUBCOMMAND: 
    all         Prints the full configuration dump, all sections included
    media       Prints only the media section
    event       ...
    ptz         ...
    device      ...
```

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
- [github.com/jfsmig/onvif/ws-discovery](https://pkg.go.dev/github.com/jfsmig/onvif/ws-discovery)
  implements the probing of the LAN network interfaces. Please refer to the CLI tools `onvif-cli discover NIC`

### Beginner's Guide

```go
params := networking.ClientParams{
    Xaddr:      "",
    Username:   os.Getenv("ONVIF_USERNAME"),
    Password:   os.Getenv("ONVIF_PASSWORD"),
    HttpClient: nil,
}
sdkDev, err := sdk.NewDevice(params)
if err != nil { /* Not a reachable OnVif device */ }
```

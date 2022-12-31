# Go OnVif Client

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/jfsmig/onvif/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/jfsmig/onvif/tree/master)
[![CodeQL](https://github.com/jfsmig/onvif/actions/workflows/codeql.yml/badge.svg)](https://github.com/jfsmig/onvif/actions/workflows/codeql.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)

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

### Clients SDK

References
  * https://github.com/topics/onvif-client
  * https://github.com/topics/onvif-camera

Go
  * https://github.com/jfsmig/onvif
  * https://github.com/use-go/onvif
  * https://github.com/yakovlevdmv/goonvif

Python
  * https://github.com/quatanium/python-onvif
  * https://github.com/abhi40308/onvif-django-client

C
  * https://github.com/mpromonet/v4l2onvif
  * https://github.com/RichardoMrMu/gsoap-onvif
  * https://github.com/torturelabs/monvif
  * https://github.com/Quedale/OnvifDeviceManager

Rust

Swift
  * https://github.com/ms2138/ONVIFDiscovery
  * https://github.com/ms2138/DahuaEvents
  * https://github.com/rvi/ONVIFCamera

Javascript
  * https://github.com/patrickmichalina/onvif-rx
  * https://github.com/ampretia/onvif-mqtt
  * https://github.com/snow-tree/camera-probe

PHP
  * https://github.com/mapbuh/onvif-client-php

### Client Apps_

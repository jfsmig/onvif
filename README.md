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
                           The URIs carry no credentials, even when the camera answered
                           with some embedded: they are printed on stdout, and a password
                           does not belong there. Supply your own at the point of use —
                           `ffplay -rtsp_transport tcp rtsp://user:pass@HOST/path`.
  subscribe TARGET...      hold an ONVIF real-time pull-point subscription on each named
                           camera and print one JSON object per line, one line per event,
                           until interrupted. Ctrl-C or SIGTERM ends the run successfully.
                           TARGET takes the same two forms as `dump`, and every identifier
                           named is resolved from a single LAN probe however many there are.
                           There is no -a: this command authenticates to every target.
  dump SUBCOMMAND TARGET   print a single JSON object holding a configuration dump of
                           the given camera. TARGET is either the camera's IP:PORT — the
                           XADDR column of `discover` — or its WS-Discovery identifier
                           urn:uuid:<uuid>, the UUID column. The identifier form probes
                           the LAN first, so it takes a few seconds longer and finds only
                           a camera that answers discovery on a non-virtual interface. It
                           is also the only form that can select a per-camera credentials
                           file; see "Per-camera credentials" below.

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

OPTIONS:
  -a, --all       belongs to `discover` and `streams`, and follows the command
                  (`onvif-cli discover -a`, never the other way round). Probe every
                  interface that is up and not loopback. By default a candidate must
                  also be multicast-capable — WS-Discovery is multicast-only — and must
                  not be one of the well-known virtual devices (docker0, br-<id>, veth*,
                  virbr*, cali*, flannel.*, ...). An interface that is down or loopback
                  is never probed either way.
                  A container's eth0 is not in that table, so probing from inside a
                  container needs no option.

      --basedir DIR
                  belongs to every command and may be given before or after it. Read the
                  per-camera credential files from DIR/credentials/*.json instead of the
                  default search path. A DIR that does not exist is an error, whereas a
                  missing default directory is not. See "Per-camera credentials" below.

  -v, --verbose   belongs to every command, and may be repeated. Diagnostics go to stderr;
                  by default only warnings and errors are printed, so a run in which
                  everything worked says nothing there at all.
                    -v    what the tool is doing — which interfaces it is probing, and
                          when it is waiting out a discovery window
                    -vv   what it loaded and which credentials it chose for each camera,
                          naming the file or the environment but never the credential
                    -vvv  the wire: the probe, and every per-call failure the SDK swallows
                          — where a 401 behind an empty section of a dump becomes visible

ENVIRONMENT:
  ONVIF_BASEDIR   base directory of the credential files, used when --basedir is not
                  given. Set but missing is an error.
  ONVIF_USERNAME  credential for a camera that no credentials file names, "admin" when
                  unset
  ONVIF_PASSWORD  credential for a camera that no credentials file names, "admin" when
                  unset
```

Data goes to stdout and diagnostics to stderr, so `onvif-cli dump all IP:PORT | jq` works
as it looks. Most commands accept aliases — `find` for `discover`, `events` for `event`,
`prof` for `profile` — which `onvif-cli COMMAND --help` lists.

`subscribe` is the one command that streams rather than returning, and so the one with no
deadline of its own: `discover`, `streams` and `dump` each give up after a minute, while a
subscription runs until it is stopped. One line per event, so
`onvif-cli subscribe urn:uuid:… | jq -c 'select(.topic | test("MotionAlarm"))'` reads as it
looks, and a whole fleet is one pipeline:

```console
onvif-cli discover | awk '$3 != "-" { print $3 }' | xargs onvif-cli subscribe
```

The record's own keys are lower case — `time`, `received`, `xaddr`, `uuid`, `topic`,
`operation`, `source`, `key`, `data` — while the names inside `source`, `key` and `data` are
the device's own, spelled as the message description publishes them (ONVIF Core §9.4.1).
Every value is a string: `Value` is `xs:anySimpleType`, so `ObjectId` is `"15"` on every
camera rather than a number on some and a string on others, and one `jq` filter works
everywhere. `time` is the stamp the camera put on the event and `received` is the collector's
own, which matters because camera clocks drift. `uuid` is *absent*, not `"-"`, when the
camera was named by its address: the `discover` column pads to keep a constant field count,
and JSON has no column to pad.

```json
{"time":"2008-10-10T12:24:57.321Z","received":"2026-09-10T14:02:32.118Z","xaddr":"192.168.1.70:80","uuid":"urn:uuid:00000700-0013-0008-0203-ec71db76e907","topic":"tns1:RuleEngine/LineDetector/Crossed","source":{"Rule":"MyImportantFence1","VideoAnalyticsConfigurationToken":"2","VideoSourceConfigurationToken":"1"},"data":{"ObjectId":"15"}}
```

A camera that drops its pull point is resubscribed, with a back-off; there is no `Renew`,
because `PullMessages` is its own keep-alive (ONVIF Core §9.1.1). A camera that never
answered is named on stderr and left out. The run fails, at once, only when not one of the
named cameras could be subscribed — an empty stream would otherwise say the same thing as a
quiet fleet. After that, stopping it is a success whatever happened to individual cameras in
between, which is what makes the exit status usable in a supervisor.

The default probe set matters on a host running containers, where the virtual interfaces
outnumber the real one by an order of magnitude. Each of them cost a socket, a multicast
join and a line of diagnostics, and none of them can reach a camera.

It also matters for `streams`, and that is worth stating plainly. WS-Discovery replies are
unauthenticated, and `streams` sends the resolved credentials to whatever answered, over
plain HTTP. On the container and VM links the peer that answers is a container or a guest
rather than a camera, so `streams -a` on such a host lets anything running there collect
those credentials. Prefer `-a` on `discover`, which authenticates nothing.

Per-camera credential files do not fix this, and in one respect make it worse: any host on
the link can claim any endpoint reference, and the endpoint reference is what *selects* the
file. A container echoing a camera's `urn:uuid:` is therefore offered that camera's own
password rather than a shared default. The files are a way of not giving every camera the
same account; they are not an authentication of the peer, and nothing here is, since ONVIF
over plain HTTP has no way to be.

The identifier itself is not a secret: ONVIF Core §8.4.9 puts `GetEndpointReference` in the
PRE_AUTH access class, so any unauthenticated client on the link may ask a device for it,
and §7.3.2 puts it in the unauthenticated multicast announcements besides. That is why it
appears in log lines and error messages here.

## Per-camera credentials

Cameras rarely share one account. `onvif-cli` resolves the credentials of each camera
separately, from JSON files under a base directory:

```console
$BASEDIR/credentials/*.json
```

`$BASEDIR` is the first of:

1. the `--basedir` argument,
2. `$ONVIF_BASEDIR`,
3. `~/.onvif`, then `/etc/onvif`.

The flag and the variable **replace** that chain rather than being prepended to it: naming
a directory means that directory, and a directory named this way that does not exist is an
error. The two default locations are searched per camera — an entry in `~/.onvif` wins,
and a camera only `/etc/onvif` knows about is still found — and either being absent is
normal.

Each file maps a camera identifier to an account:

```json
{
  "urn:uuid:00000700-0013-0008-0203-ec71db76e907": {
    "user": "admin",
    "password": "REDACTED"
  }
}
```

The identifier is the one `discover` prints in its UUID column — the WS-Discovery endpoint
reference, which ONVIF Core §7.3.1 requires to be "stable, globally unique … and constant
across network interfaces", which is what makes it the right key. It is the key inside the
file that identifies the camera, not the file name, so a file may hold a whole fleet;
naming each file `<uuid>.json` is just what makes `ls` readable. Both the `urn:uuid:` form
§7.1 asks for and the shorter `uuid:` some firmware emits are accepted, in either case —
§7.3.1 says a device *should* use the first, not that it shall, so both are conformant.

Credentials are then resolved in this order:

1. the entry for that camera, in the first directory that has one,
2. `ONVIF_USERNAME` / `ONVIF_PASSWORD`, which cover every camera of the run — the usual
   shape of a local fleet with one account,
3. the built-in `admin` / `admin`.

Which of the three answered is named on stderr, and in the error when the connection to a
camera fails, because "401" and "401 with the compiled-in `admin`/`admin`" call for
different next steps. The credentials themselves are never logged. Note that a call
rejected *after* the connection succeeds leaves its part of the dump empty, which is how
`sdk` treats every per-call failure — so an empty section is worth a second run with
`-vvv`, which prints the rejection that produced it.

Two consequences worth knowing:

- A camera named by its address alone has no identifier, so `dump all IP:PORT` always uses
  the blanket credentials. Pass `dump all urn:uuid:…` to select a file.
- `chmod 600` the files. The tool warns on stderr when one is readable by other local
  accounts; it does not refuse to use it.

## SDK

A High Level go package aims at fetching information from the devices:
- [github.com/jfsmig/onvif/sdk](https://pkg.go.dev/github.com/jfsmig/onvif/sdk)

Low-Level go packages implement the OnVIF unitary SOAP calls. For each call :
- [github.com/jfsmig/onvif/device](https://pkg.go.dev/github.com/jfsmig/onvif/device)
- [github.com/jfsmig/onvif/event](https://pkg.go.dev/github.com/jfsmig/onvif/event)
- [github.com/jfsmig/onvif/ptz](https://pkg.go.dev/github.com/jfsmig/onvif/ptz)
- [github.com/jfsmig/onvif/media](https://pkg.go.dev/github.com/jfsmig/onvif/media)

Two more packages carry the request and reply **types only** — they have no `calls.txt` and
so no `Call_*` wrappers, which means their operations cannot be issued yet:
- [github.com/jfsmig/onvif/imaging](https://pkg.go.dev/github.com/jfsmig/onvif/imaging)
- [github.com/jfsmig/onvif/analytics](https://pkg.go.dev/github.com/jfsmig/onvif/analytics)

`imaging` and `analytics` are already among the service names `networking` will route, so
what is missing is the wrappers rather than the plumbing.

> **Breaking change.** The directory was `Imaging/` until it was renamed to `imaging/`, so the
> import path `github.com/jfsmig/onvif/Imaging` is gone and is spelled `…/imaging` now. It was
> the only capitalised package here, and the mismatch was not cosmetic: `onvif-codegen` refuses
> to generate into a directory whose name differs from its package clause, so the sentence
> above was false for `Imaging` — the wrappers could never have been added without this rename.
> The package exports request and reply types only and has no callable operation, so an
> importer's fix is the path and nothing else.

Helpers:
- [github.com/jfsmig/onvif/credentials](https://pkg.go.dev/github.com/jfsmig/onvif/credentials)
  answers "which credentials for the camera bearing this identifier?". `credentials.Resolver`
  is the interface; a `Store` reads the `*.json` files described above, `Static` is a blanket
  credential, and `Chain` states the precedence between them. An application that already
  holds its credentials in memory — loaded from a vault or a database at start-up —
  implements the interface and keeps the rest of the tool unchanged. `Resolve` answers from
  memory and cannot fail, so a source that does I/O per lookup belongs behind a type that
  loads eagerly, as `Store` does
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

# AGENTS.md

## What this is

A Go client for IP cameras speaking the [ONVIF](https://www.onvif.org/) protocol — ONVIF
requests are SOAP 1.2 over HTTP, authenticated with a WS-Security `UsernameToken`.

The repository is a long-diverged fork of [goonvif](https://github.com/use-go/goonvif),
later `use-go/onvif`; the link to upstream was cut. That history is why most files carry a
provenance notice, and why `LICENSE.MIT` exists alongside the AGPL `LICENSE`.

It is laid out in three layers, low to high:

| Package | Role |
| --- | --- |
| `networking/` | The SOAP client. `Client.CallMethod` marshals a request struct, wraps it in an envelope, adds WS-Security, POSTs it. `ReadAndParse` unmarshals the reply. |
| `device/` `media/` `ptz/` `event/` | One `Call_<Method>` wrapper per ONVIF operation — **generated**, see below — plus a hand-written `types.go` holding the request/reply structs. |
| `sdk/` | The high-level layer. `sdk.Appliance` is the interface a caller wants: `NewDevice(ctx, info, auth, httpClient)` connects and loads the service endpoints, then `FetchMedia`, `FetchPTZ`, `FetchDeviceNetwork`, `FetchProfiles`… return fully-hydrated structs. |

Supporting packages: `xsd/` (XSD primitives and the generated ONVIF schema types),
`Imaging/` and `analytics/` (types only), `utils/` (two sentinel errors, `ErrHTTP` and
`ErrNotOnvif`).

Discovery is **not** in this repository. It lives in `github.com/jfsmig/go-wsd`, which
also provides the `gosoap` envelope builder this project uses.

`bin/` is source, not build output: `bin/onvif-cli` is the CLI (`discover`, `streams`,
`dump <section> IP:PORT`, credentials from `ONVIF_USERNAME` / `ONVIF_PASSWORD`), and
`bin/onvif-codegen` is the generator.

## Commands

```sh
go build ./...
go vet ./...
go test -race ./...              # only utils/ is covered so far; keep it green
gofmt -l .                       # must print nothing
go generate ./... && git diff --quiet --exit-code   # generated files must match the template
go run ./bin/onvif-cli discover  # smoke-test against the LAN
```

`.circleci/config.yml` gates on `go install`, `go test`, `go vet`, and that last
generate-then-diff check. There is no test suite; a change is verified by building and by
running the CLI against a real camera.

## The generator — read this before touching `device/`, `media/`, `ptz/`, `event/`

205 of the 231 `.go` files are generated. **Never hand-edit a `*_auto.go` file**; CI
regenerates them and diffs, so an edit is reverted and the build fails.

The pipeline: each package has a `calls.txt` (one ONVIF method name per line, `#`
comments allowed) and a `//go:generate` line in its `types.go`. `bin/onvif-codegen`
expands one template — the `mainTemplate` const in `bin/onvif-codegen/sdk.go` — once per
entry, writing `<Method>_auto.go`.

- To add an operation: add its request/reply structs to `types.go`, add the name to
  `calls.txt`, run `go generate ./...`.
- To change the shape of every wrapper, or its licence header: edit `mainTemplate`, then
  regenerate. Editing the template without regenerating breaks CI.
- Counts must stay 1:1 — `device` 89, `media` 79, `ptz` 28, `event` 9.

## Licence header on every `.go` file

AGPL-3.0-or-later. Copy the 14-line notice from an existing file; `utils/errorz.go` is the
plain case. Twelve files carry an extra provenance block between the copyright line and
the AGPL block, because they still contain upstream MIT code:

| Files | Extra provenance block |
| --- | --- |
| `doc.go`, `Imaging/types.go`, `analytics/types.go`, `device/types.go`, `media/types.go`, `ptz/types.go`, `event/types.go`, `event/operation.go`, `networking/networking.go`, `xsd/built_in.go`, `xsd/onvif/onvif.go`, `xsd/iso8601/iso8601_duration.go` | derives from goonvif / use-go/onvif, MIT, see `LICENSE.MIT`, © 2018 Yakovlev Dmitry, Zhorzh Palanjyan, Crazybber |
| everything else, generated files included | none |

**Keep the blank line between the notice and `package X`.** Without it Go takes the licence
as the package doc comment — silently. `doc.go` is the file to look at: notice, blank line,
package doc, `package onvif`.

`docs/` is out of scope for all of this. It holds ONVIF's own specification and WSDL
files, which carry ONVIF's terms — *"No license is granted to modify this document"* — as
do the `xsd/onvif/*.xsd` schemas. Do not add headers to them and do not edit them.

## Conventions and traps

- **The endpoint is chosen by the request struct's package name.** `CallMethod` does
  `reflect.TypeOf(method).PkgPath()`, takes the last segment, lowercases it, and looks it
  up in the endpoint map (`networking/client.go:127-131`). So the directory names
  `device`, `media`, `ptz`, `event` are load-bearing: renaming one silently routes its
  calls to the wrong service, or to none. A new service package must be named after its
  ONVIF endpoint.

- **`sdk` swallows per-call errors by design.** The `Fetch*` methods return a struct, not
  `(struct, error)`; a failed sub-call is logged at trace level and leaves its field
  zero, so one unsupported operation cannot lose a whole dump. Keep that shape — cameras
  vary wildly in what they implement.

- **Fan out with `sync.WaitGroup.Go`**, as `sdk/profiles.go`, `bin/onvif-cli/dump.go` and
  `bin/onvif-cli/discover.go` do. Independent fetches against one camera, or probes across
  interfaces, should not be sequential: each costs a network round trip and the CLI runs
  under a one-minute context.

- **Do not reach for `errgroup`.** Its first-error-cancels-the-rest semantics is wrong
  here: a camera lacking PTZ errors on every PTZ call, which is normal, and cancelling the
  siblings would turn a partial result into an empty one. The fan-out closures handle
  their own errors and return nothing, so `WaitGroup` states the intent exactly.

- Prefer short functions. Comment functions rather than lines, explain *why* rather than
  restating the code, and cite the clause when behaviour comes from a spec (`ONVIF Core
  section 7.3.6`, `SOAP 1.2 Part 1 section 5.2.3`). Always comment in English.

- No dead code, no commented-out code, no ignored errors.

- **Never `log.Print*` to the standard logger.** Diagnostics go through a replaceable
  logger: `sdk` exports a package-level `zerolog` `Logger` (`sdk/appliance.go:36`) that an
  application can swap out, and the CLI has its own in `bin/onvif-cli/main.go`. Packages
  below `sdk` — `networking`, `xsd`, `utils` — return errors and log nothing. The one
  breach is `xsd/built_in.go:218`, which calls `log.Fatalln` and so kills the caller's
  process from inside a library; do not copy it, and fix it if you touch that function.


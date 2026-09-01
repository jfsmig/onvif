# ONVIF reference documents

Third-party ONVIF documents, kept here so that the choices made in this repository can cite a
clause rather than an assertion. They are reproduced under ONVIF's own terms ("No license is
granted to modify this document") and are **not** covered by the AGPL that applies to the rest
of the repository. Do not edit them and do not add licence headers to them.

## Core

| Document | Version | Date |
| --- | --- | --- |
| [Core Specification](./ONVIF-Core-Specification.pdf) | 19.12 | December 2019 |

Defines the services, the SOAP binding, WS-Security, error handling and discovery. It does
**not** define which operations make up a Profile — that is what the documents below are for.

## Profiles

The Profile specifications state, per Profile, which features and operations a device and a
client must or may implement. Retrieved 2026-09-01 from `https://www.onvif.org/profiles/`.

| Profile | Document | Version | Date | Scope |
| --- | --- | --- | --- | --- |
| S | [Profile S](./ONVIF-Profile-S-Specification.pdf) | 1.3 | November 2019 | video streaming, PTZ, audio |
| T | [Profile T](./ONVIF-Profile-T-Specification.pdf) | 1.0 | September 2018 | advanced streaming, H.264/H.265, metadata |
| G | [Profile G](./ONVIF-Profile-G-Specification.pdf) | 1.1 | October 2025 | edge storage, recording, search, replay |
| M | [Profile M](./ONVIF-Profile-M-Specification.pdf) | 1.1 | March 2024 | metadata and analytics |
| C | [Profile C](./ONVIF-Profile-C-Specification.pdf) | 1.0 | December 2013 | door control, access control events |
| A | [Profile A](./ONVIF-Profile-A-Specification.pdf) | 1.0 | June 2017 | access control configuration |
| D | [Profile D](./ONVIF-Profile-D-Specification.pdf) | 1.0 | June 2021 | access control peripherals |

Plus a cross-Profile comparison, useful for seeing at a glance which Profile owns a feature:

| Document | Version | Date |
| --- | --- | --- |
| [Profile Feature Overview](./ONVIF-Profile-Feature-Overview.pdf) | 2.6 | April 2022 |

Two Profiles are absent, deliberately:

- **Profile V** is a release candidate at the time of retrieval and is published only to
  eligible ONVIF members, so there is no public document to vendor.
- **Profile Q** was deprecated on 1 April 2022 and is no longer listed.

## Service definitions

[`wsdl/`](./wsdl) holds the WSDL for each ONVIF service, which is the authority for the
message shapes: every `xml:` tag in the request and reply structs is checked against the
`portType` and inline schema of the matching file. `onvif.xsd`, the common schema those WSDLs
import as `../../../ver10/schema/onvif.xsd`, is **not** vendored — so the types in
`xsd/onvif/` cannot currently be machine-checked against it.

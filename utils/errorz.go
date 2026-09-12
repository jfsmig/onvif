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

package utils

// constError is a string that satisfies error. Being a defined string type, it can be
// declared const: a sentinel that no importer can reassign, and that stays comparable so
// errors.Is keeps working through a wrapping chain.
type constError string

func (e constError) Error() string { return string(e) }

const (
	// ErrHTTP reports a non-200 reply from the device. Callers wrap it with the status,
	// so match it with errors.Is rather than ==.
	ErrHTTP = constError("http request error")

	// ErrNotOnvif reports a host that answered but does not speak ONVIF.
	ErrNotOnvif = constError("not an ONVIF device")

	// ErrSOAPFault reports that the device answered with a SOAP 1.2 fault rather than the
	// response the operation declares. Callers wrap it with the code, the subcode and the
	// reason, so match it with errors.Is rather than ==.
	//
	// It is separate from ErrHTTP because the two answer different questions and a reply can
	// be both. ONVIF Core section 5.11.2.1 makes the fault the only channel for an operation
	// error and section 5.11.2.2 makes the ter: subcode the discriminator, while the HTTP
	// status only says which half of SOAP 1.2's binding the fault fell into -- 400 for
	// env:Sender, 500 for env:Receiver. A device that answers a fault with 200, which is not
	// conformant and is common, yields this and no ErrHTTP at all.
	ErrSOAPFault = constError("soap fault")

	// ErrNotAuthorized reports that the device rejected our credentials. It is wrapped
	// alongside ErrSOAPFault or ErrHTTP rather than instead of them, so matching either of
	// those keeps working.
	//
	// It exists because this one cause has to be told from every other per-call failure.
	// ONVIF makes whole services and many operations conditional, so a fault usually means
	// "this camera does not do that" -- which is an answer about the camera, and which sdk
	// deliberately swallows into an empty field. A rejected credential is not an answer about
	// the camera at all: every other call will fail the same way, and the empty result an
	// operator is left holding says nothing about why. ONVIF Core section 5.11.2.2 Table 5
	// gives it the subcode ter:NotAuthorized; the HTTP binding gives it 401 or 403.
	ErrNotAuthorized = constError("not authorized")

	// ErrDigestOffered reports that the device's rejection carried an HTTP Digest challenge.
	// It is wrapped alongside ErrNotAuthorized, never instead of it, for the same reason
	// ErrNotAuthorized is wrapped alongside ErrHTTP: a caller matching the broader cause must
	// keep matching.
	//
	// Read it as "the device also speaks digest", not as "the device speaks only digest".
	// Every camera on the development bench sends this challenge on a refused password and
	// then accepts WS-UsernameToken once the password is right, so it does not tell a wrong
	// credential apart from a scheme this library cannot satisfy. It cannot: WS-UsernameToken
	// is not an HTTP authentication scheme, so a device that supports it has nothing to say
	// about it in WWW-Authenticate.
	//
	// It is still worth surfacing, because ONVIF Core 5.12.1 makes digest the scheme a device
	// shall be protected with and WS-UsernameToken the legacy exception, and only the
	// exception is implemented here -- see the header of sdk/profiles/S.profile. So on a
	// device that has dropped the exception, this is the only hint available. RFC 7235
	// section 4.1 puts the scheme in the WWW-Authenticate header, which is where it is read
	// from.
	ErrDigestOffered = constError("device offered HTTP digest authentication")

	// ErrNoService reports that the appliance advertises no endpoint for the service a
	// request was addressed to. It is a property of the device, not a failure of the
	// exchange: nothing was sent. Callers need to tell it apart from a rejected request
	// because ONVIF makes whole services conditional — a camera without PTZ answers
	// everything else perfectly well — so this is the error a conditional capability
	// yields, and it must not be reported as a fault.
	ErrNoService = constError("no endpoint for the service")
)

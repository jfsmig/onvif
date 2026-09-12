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

	// ErrNoService reports that the appliance advertises no endpoint for the service a
	// request was addressed to. It is a property of the device, not a failure of the
	// exchange: nothing was sent. Callers need to tell it apart from a rejected request
	// because ONVIF makes whole services conditional — a camera without PTZ answers
	// everything else perfectly well — so this is the error a conditional capability
	// yields, and it must not be reported as a fault.
	ErrNoService = constError("no endpoint for the service")
)

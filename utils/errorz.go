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

	// ErrNoService reports that the appliance advertises no endpoint for the service a
	// request was addressed to. It is a property of the device, not a failure of the
	// exchange: nothing was sent. Callers need to tell it apart from a rejected request
	// because ONVIF makes whole services conditional — a camera without PTZ answers
	// everything else perfectly well — so this is the error a conditional capability
	// yields, and it must not be reported as a fault.
	ErrNoService = constError("no endpoint for the service")
)

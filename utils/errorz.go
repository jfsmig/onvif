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
)

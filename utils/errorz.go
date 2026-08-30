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

type constError string

func (e constError) Error() string { return string(e) }

var ErrUnreachable = constError("unreachable device")
var ErrHttp = constError("http request error")
var ErrNotOnvif = constError("not an OnVif device")
var ErrUnsupportedPTZ = constError("unsupported PTZ")
var ErrUnsupportedCall = constError("unsupported call")

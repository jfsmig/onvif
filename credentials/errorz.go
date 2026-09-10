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

package credentials

// constError is the idiom of utils/errorz.go: a defined string type, so the sentinels
// below can be const -- no importer can reassign them -- while staying comparable, so
// errors.Is keeps working through a wrapping chain.
type constError string

func (e constError) Error() string { return string(e) }

const (
	// ErrMalformedFile reports a credentials file that could not be turned into entries:
	// invalid JSON, an empty file, an empty camera identifier, or an entry with no user.
	// It is wrapped with the file name and never with any of the file's content -- see
	// jsonFault for why the decoder's own message does not qualify.
	ErrMalformedFile = constError("malformed credentials file")

	// ErrDuplicateEntry reports one camera claimed twice in the same directory. An error
	// rather than a first-wins rule because the two entries may disagree, and settling
	// that by the order fs.ReadDir happens to return would make the effective password a
	// function of the file names.
	ErrDuplicateEntry = constError("duplicate camera identifier")
)

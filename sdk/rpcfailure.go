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

package sdk

import (
	"errors"

	"github.com/rs/zerolog"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/utils"
)

// rpcFailure is the one place that decides how loudly a swallowed per-call failure is
// reported. Every Fetch* closure logs through it.
//
// Trace for almost all of them, which is the Fetch* contract doing its job: ONVIF makes whole
// services and many operations conditional, so a fault is usually the camera describing
// itself, the zero field left behind is the truthful record of that, and an operator who
// wants the detail asks for it with -vvv.
//
// Warn for one cause only: the device rejected our credentials. That is not a description of
// the camera at all. Every other call will fail the same way, so what the operator is left
// holding -- an empty section of a dump, a missing line of `streams`, exit 0 either way --
// says nothing whatever about why. bin/onvif-cli promises that warnings and errors print
// whatever the verbosity count, and this is the case that promise was written for: its own
// -vv documentation calls the credential choice "the pair of questions a 401 raises", which
// is hard to ask when nothing says a 401 happened.
//
// Once per appliance, never once per call, which is the other half of it. A single `dump all`
// against a camera with the wrong password produced 41 authentication faults; every one true,
// and forty of them noise that would bury the first. networking.Client.NoteAuthRejected
// answers true exactly once, to exactly one of the goroutines racing to discover it.
func rpcFailure(client *networking.Client, err error, rpc string) *zerolog.Event {
	// A line of its own rather than the same one at a higher level, because the caller's
	// Msg is a section label -- "profile", "device", "audio" -- chosen to group trace output
	// and not to be read on its own. An operator meeting this at default verbosity needs a
	// sentence. The trace line below is still emitted, unchanged, so -vvv shows exactly what
	// it always did.
	if errors.Is(err, utils.ErrNotAuthorized) && client.NoteAuthRejected() {
		Logger.Warn().Err(err).Str("addr", client.Xaddr()).Str("rpc", rpc).
			Msg(authAdvice(err))
	}
	return Logger.Trace().Err(err).Str("rpc", rpc)
}

// authAdvice picks the sentence that tells the operator what to change.
//
// The two cases need opposite actions, and the wrong one wastes real time. A refused password
// is fixed by supplying another. A device that asked for HTTP Digest refuses every password
// this library can offer, because ONVIF Core 5.12.1 makes digest the required scheme and
// WS-UsernameToken the legacy exception, and only the exception is implemented here -- see
// the header of sdk/profiles/S.profile. Reporting that as a rejected credential sends an
// operator round a loop of rotations that cannot end.
//
// Deliberately not a third case for "both": a device that offers digest alongside something
// else still gets this sentence, because what the operator needs to know is that the scheme
// is in play at all.
func authAdvice(err error) string {
	if errors.Is(err, utils.ErrDigestRequired) {
		return "Camera asked for HTTP digest authentication, which this library does not " +
			"speak; the credentials may be correct and whatever it reports will be incomplete"
	}
	return "Camera rejected the credentials, whatever it reports will be incomplete"
}

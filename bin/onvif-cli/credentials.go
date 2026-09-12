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

package main

import (
	"os"
	"path/filepath"

	"github.com/jfsmig/onvif/v2/credentials"
	"github.com/jfsmig/onvif/v2/networking"
)

// The base directories searched when neither --basedir nor ONVIF_BASEDIR names one. The
// per-user location comes first: a workstation's own file must win over what the machine
// was provisioned with.
const (
	userBaseDir   = ".onvif"
	systemBaseDir = "/etc/onvif"
)

// The labels a credential's origin is reported by. They reach a log line and an error
// message, never a credential: "authentication failed" and "authentication failed with the
// compiled-in admin/admin" call for different next actions, and today the operator cannot
// tell them apart.
const (
	sourceEnvironment = "environment"
	sourceBuiltIn     = "built-in default"
)

// credentialsSearchPath returns the base directories to search, most significant first,
// and whether the first of them was named by the operator.
//
// That flag is the whole difference in error handling: a directory somebody named and that
// does not exist is a mistake worth stopping for, while a default one that does not exist
// is the normal state of a host nobody has configured.
//
// The flag and the variable REPLACE the chain rather than being prepended to it. An
// operator who names a directory means that directory, and quietly falling through to
// /etc/onvif behind their back is the kind of surprise that ends in a support thread.
//
// Pure over its arguments, like probeableInterfaceNames, so that the whole policy is
// testable without a HOME, an environment or a filesystem.
func credentialsSearchPath(flagDir, envBaseDir, home string) (bases []string, explicit bool) {
	if flagDir != "" {
		return []string{flagDir}, true
	}
	if envBaseDir != "" {
		return []string{envBaseDir}, true
	}
	// The home directory has to be absolute, and testing that is not belt and braces:
	// os.UserHomeDir returns $HOME verbatim on Unix, and filepath.Join with a relative or
	// empty home yields a relative path -- so a unit started without HOME, a systemd
	// service or a scratch container, would read its credentials out of whatever
	// directory it happened to be started in.
	if filepath.IsAbs(home) {
		bases = append(bases, filepath.Join(home, userBaseDir))
	}
	return append(bases, systemBaseDir), false
}

// buildResolver assembles the precedence the tool documents: the file entry for that
// camera, then the ONVIF_USERNAME / ONVIF_PASSWORD blanket a fleet sharing one account
// sets, then the compiled-in default.
//
// It reads every store eagerly, so a mistyped path or a malformed file is reported once,
// here, before any camera is contacted -- and so the resolver it returns is immutable and
// safe to share with whatever goroutine ends up asking it.
func buildResolver(flagDir string) (credentials.Resolver, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		// Not fatal: it only means the per-user location is unknown, and the system one
		// is still there. The error itself is not worth a line, the consequence is.
		Logger.Debug().Msg("no home directory, looking for credentials in " + systemBaseDir + " only")
		home = ""
	}

	bases, explicit := credentialsSearchPath(flagDir, os.Getenv("ONVIF_BASEDIR"), home)

	var chain credentials.Chain
	for _, base := range bases {
		store, err := loadBase(base, explicit)
		if err != nil {
			return nil, err
		}
		// A nil store is a base directory that does not exist. It is dropped here rather
		// than appended: a nil *Store put into a []Resolver does not compare equal to nil,
		// so Chain's own nil check would not catch it, and only (*Store).Resolve's nil
		// receiver would keep the run alive.
		if store != nil {
			chain = append(chain, store)
		}
	}

	source, auth := environmentCredentials()
	return append(chain, credentials.Static(source, auth)), nil
}

// loadBase reads one base directory, tolerating its absence only when it is a default: a
// directory somebody named and that is not there is a mistake worth stopping for, while a
// default one that is not there is the normal state of an unconfigured host.
func loadBase(base string, explicit bool) (*credentials.Store, error) {
	load := credentials.LoadBaseIfPresent
	if explicit {
		load = credentials.LoadBase
	}
	store, err := load(base)
	if err != nil {
		return nil, err
	}
	reportStore(base, store)
	return store, nil
}

// reportStore says what was loaded and warns about what any local account can read.
//
// The warning is one line naming every offending file, in the shape warnNothingToProbe
// uses, rather than one line per file: an operator who has ten cameras has ten files, and
// ten warnings would be scrolled past. It warns and never refuses -- every file in
// ~/.onvif/credentials is 0664 on the machine this was written on, and a tool that stopped
// working the day it learnt to read them would simply not be used.
func reportStore(base string, store *credentials.Store) {
	if store == nil {
		Logger.Debug().Str("base", base).Msg("no credentials directory there")
		return
	}
	Logger.Debug().Str("base", base).Int("entries", store.Len()).Msg("credentials loaded")
	if lax := store.LaxPermissions(); len(lax) > 0 {
		Logger.Warn().Strs("files", lax).
			Msg("credentials files are readable by other local accounts, consider chmod 600")
	}
}

// environmentCredentials reads the blanket variables. blanketCredentials decides what they
// mean, kept pure over its arguments for the same reason credentialsSearchPath is.
func environmentCredentials() (string, networking.ClientAuth) {
	user := os.Getenv("ONVIF_USERNAME")
	password, passwordSet := os.LookupEnv("ONVIF_PASSWORD")
	return blanketCredentials(user, password, passwordSet)
}

// blanketCredentials is what ONVIF_USERNAME and ONVIF_PASSWORD mean: "this account covers
// the cameras this run reaches", which is the usual shape of a local fleet. The admin/admin
// default is kept because it is what the tool has always done, but it is labelled apart so
// that an authentication failure can say which of the two was tried.
//
// The label names the credential that is actually sent, not the variable that happened to
// exist. `export ONVIF_USERNAME="$CAM_USER"` with an unset CAM_USER exports an empty
// string, and reporting that as "environment" while sending the compiled-in admin/admin is
// the one lie this label must not tell.
//
// The two variables are not symmetric, and that is deliberate: an empty password is a
// legitimate ONVIF account and networking still sends a UsernameToken for it, whereas an
// empty username sends no token at all (networking/client.go:239-253, pinned by
// networking/wssecurity_test.go), so it cannot be a credential and falls back in the label
// as well as in the value.
func blanketCredentials(user, password string, passwordSet bool) (string, networking.ClientAuth) {
	source := sourceBuiltIn
	if user != "" || passwordSet {
		source = sourceEnvironment
	}
	if user == "" {
		user = "admin"
	}
	if !passwordSet {
		password = "admin"
	}
	return source, networking.ClientAuth{Username: user, Password: password}
}

// credentialsFor answers for one camera, and says where the answer came from.
//
// A camera reached by its address alone carries no identifier into this lookup: discovery
// is what hands us one, and it runs before any client exists. ONVIF Core section 8.4.9
// does define GetEndpointReference, whose access class is PRE_AUTH and which would
// therefore yield the identifier from the address alone -- but a device only *should*
// support it, and the call would have to precede sdk.NewDevice. Until it does, an address
// lands on the blanket, which is why `dump` accepts the urn:uuid: form at all.
//
// Nothing about the credential itself is ever logged, not even the username: AGENTS.md
// forbids logging a ClientAuth, and leaving one field of it loggable would put the next
// reader one careless edit away from the other.
func credentialsFor(uuid string) (networking.ClientAuth, string) {
	if resolver == nil {
		// Cobra runs its completion machinery without the root's PersistentPreRunE, so a
		// future ValidArgsFunction completing camera identifiers would reach here with
		// nothing built. A nil interface would panic; this is a miss instead.
		return networking.ClientAuth{}, "none"
	}
	auth, source, ok := resolver.Resolve(uuid)
	if !ok {
		// Only reachable if the chain lost its Static tail, which always matches. The
		// empty ClientAuth that goes back sends no UsernameToken at all
		// (networking/client.go:239-253), so the request is issued unauthenticated -- and
		// the "none" label is what makes the resulting 401 say so instead of naming a
		// source that never answered.
		return networking.ClientAuth{}, "none"
	}
	if uuid == "" {
		Logger.Debug().Str("credentials", source).Msg("no camera identifier, using the blanket credentials")
	} else {
		Logger.Debug().Str("uuid", uuid).Str("credentials", source).Msg("credentials resolved")
	}
	return auth, source
}

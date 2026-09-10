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

// Where the credentials are looked for is a policy, and like the interface filter in
// interfaces.go it is written as a pure function so that the whole of it can be pinned
// without a HOME, an environment or a filesystem.
//
// Two of the cases below are the ones worth having. A flag or a variable REPLACES the
// default chain instead of being prepended to it, because an operator who names a
// directory means that directory. And an empty home yields /etc/onvif alone: the tempting
// filepath.Join(home, ".onvif") produces a *relative* path when home is empty, so a unit
// started without HOME -- a systemd service, a scratch container -- would read its
// credentials out of whatever directory it happened to be started in.

import (
	"slices"
	"testing"
)

func TestCredentialsSearchPath(t *testing.T) {
	const home = "/home/operator"

	for _, tc := range []struct {
		name         string
		flagDir      string
		envBaseDir   string
		home         string
		want         []string
		wantExplicit bool
	}{
		{
			"nothing named, the default chain",
			"", "", home,
			[]string{"/home/operator/.onvif", "/etc/onvif"}, false,
		},
		{
			"the flag replaces the chain",
			"/srv/onvif", "", home,
			[]string{"/srv/onvif"}, true,
		},
		{
			"the variable replaces the chain",
			"", "/opt/onvif", home,
			[]string{"/opt/onvif"}, true,
		},
		{
			"the flag beats the variable",
			"/srv/onvif", "/opt/onvif", home,
			[]string{"/srv/onvif"}, true,
		},
		{
			"an unset flag is not a named directory",
			"", "", home,
			[]string{"/home/operator/.onvif", "/etc/onvif"}, false,
		},
		{
			"no home: the system directory alone, never a relative path",
			"", "", "",
			[]string{"/etc/onvif"}, false,
		},
		{
			"no home, but a named directory is still honoured",
			"/srv/onvif", "", "",
			[]string{"/srv/onvif"}, true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, explicit := credentialsSearchPath(tc.flagDir, tc.envBaseDir, tc.home)
			if !slices.Equal(got, tc.want) {
				t.Errorf("credentialsSearchPath(%q, %q, %q) = %v, want %v",
					tc.flagDir, tc.envBaseDir, tc.home, got, tc.want)
			}
			if explicit != tc.wantExplicit {
				t.Errorf("explicit = %v, want %v: it is what decides whether a missing "+
					"directory is an error or the normal state of an unconfigured host",
					explicit, tc.wantExplicit)
			}
		})
	}
}

func TestCredentialsSearchPathNeverYieldsARelativePath(t *testing.T) {
	// The consequence of the join above, stated as its own assertion because it is the
	// one failure that would silently read a file the operator never wrote.
	for _, home := range []string{"", ".", "relative/home"} {
		bases, _ := credentialsSearchPath("", "", home)
		for _, base := range bases {
			if base[0] != '/' {
				t.Errorf("home %q yielded the relative base %q", home, base)
			}
		}
	}
}

func TestBlanketCredentialsLabelsWhatItActuallySends(t *testing.T) {
	// The label is the whole reason Resolve reports a source: "401" and "401 with the
	// compiled-in admin/admin" call for different next actions. The slip was to label on
	// whether the variable existed rather than on the value that survived, so an exported
	// but empty ONVIF_USERNAME -- what `export ONVIF_USERNAME="$CAM_USER"` with an unset
	// CAM_USER produces -- reported "environment" on a run that had sent the built-in
	// credentials: the one lie this label must not tell.
	//
	// The asymmetry between the two variables is deliberate and pinned below. An empty
	// password is a legitimate ONVIF account and still sends a UsernameToken; an empty
	// username sends no token at all (networking/client.go:239-253, pinned by
	// networking/wssecurity_test.go), so it cannot be a credential.
	for _, tc := range []struct {
		name                   string
		user, password         string
		passwordSet            bool
		wantSource             string
		wantUser, wantPassword string
	}{
		{"nothing set", "", "", false, sourceBuiltIn, "admin", "admin"},
		{"both set", "operator", "s3cret", true, sourceEnvironment, "operator", "s3cret"},
		{"an exported but empty username is not a credential", "", "", false, sourceBuiltIn, "admin", "admin"},
		{"the password alone: the default user, my password", "", "s3cret", true, sourceEnvironment, "admin", "s3cret"},
		{"an empty password is legitimate and comes from the environment", "operator", "", true, sourceEnvironment, "operator", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, auth := blanketCredentials(tc.user, tc.password, tc.passwordSet)
			if source != tc.wantSource {
				t.Errorf("source = %q, want %q: the label names the credential that was "+
					"sent, not the variable that happened to exist", source, tc.wantSource)
			}
			if auth.Username != tc.wantUser || auth.Password != tc.wantPassword {
				t.Errorf("credentials = %q/<redacted>, want the user %q", auth.Username, tc.wantUser)
			}
		})
	}
}

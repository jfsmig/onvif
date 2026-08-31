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

package onvif

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/jfsmig/onvif/xsd"
)

// onvif-cli JSON-encodes these structs straight to stdout, so a device that returns a
// password (which ONVIF says it must not, and which plenty of cameras do anyway) would put
// it in whatever file the operator redirected the dump into. The xml tags must survive
// untouched, because CreateUsers and SetUser legitimately carry a secret to the device.

const secret = "hunter2-SUPER-SECRET"

func TestSecretsAreNotSerialisedToJSON(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"User.Password", User{Username: "admin", Password: secret}},
		{"RemoteUser.Password", RemoteUser{Username: "admin", Password: secret}},
		{"Dot11PSKSet.Key", Dot11PSKSet{Key: Dot11PSK(secret)}},
		{"Dot11PSKSet.Passphrase", Dot11PSKSet{Passphrase: Dot11PSKPassphrase(secret)}},
		{"CertificateWithPrivateKey.PrivateKey", CertificateWithPrivateKey{PrivateKey: BinaryData{}}},
		{"EAPMethodConfiguration.Password", EAPMethodConfiguration{Password: xsd.String(secret)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			if strings.Contains(string(b), secret) {
				t.Fatalf("secret leaked into JSON: %s", b)
			}
		})
	}
}

// The redaction must not break the request path: a password still has to reach the device.
func TestSecretsStillMarshalToXML(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"User.Password", User{Username: "admin", Password: secret}},
		{"RemoteUser.Password", RemoteUser{Username: "admin", Password: secret}},
		{"EAPMethodConfiguration.Password", EAPMethodConfiguration{Password: xsd.String(secret)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := xml.Marshal(tc.value)
			if err != nil {
				t.Fatalf("xml.Marshal: %v", err)
			}
			if !strings.Contains(string(b), secret) {
				t.Fatalf("xml lost the secret, so the device would never receive it: %s", b)
			}
		})
	}
}

// Non-secret siblings must still be present, or the redaction has gone too far.
func TestNonSecretFieldsSurvive(t *testing.T) {
	b, err := json.Marshal(User{Username: "admin", Password: secret, UserLevel: UserLevel("Administrator")})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	for _, want := range []string{"admin", "Administrator"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("JSON lost %q: %s", want, b)
		}
	}
}

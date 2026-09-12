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

	"github.com/jfsmig/onvif/v2/xsd"
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

// The other side of the rule, pinned on purpose so that nobody "fixes" it.
//
// A tt:ItemList is a key/value channel whose names the device chooses (ONVIF Core section
// 9.4.1, and section 9.4.3 for the description that names them), and an ElementItem's value
// is raw device XML. There is no field here to tag, so the json:"-" mechanism does not reach
// this payload at all — and tagging the list would delete the whole of what `onvif-cli
// subscribe` and `dump media` exist to print. What has to be true instead is that nothing
// else in the process puts a credential into one of these, which is a property of the callers
// and not of this struct.
//
// It matters that this is written down: an ElementItem's value used to be dropped on
// unmarshal, so this diff widened what a redirected `dump media` can contain, and nobody
// reading the tags would infer that.
func TestAnItemListIsNotRedactedBecauseItCannotBe(t *testing.T) {
	list := ItemList{
		SimpleItem:  []SimpleItem{{Name: "Password", Value: xsd.AnySimpleType(secret)}},
		ElementItem: []ElementItem{{Name: "VendorBlob", Value: "<Password>" + secret + "</Password>"}},
	}

	b, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(b), secret) {
		t.Fatalf("an item value was dropped from JSON: %s\n"+
			"a device-named item is payload and not a field, so it cannot be redacted by "+
			"tag — if this is being redacted now, the notification payload is going with it", b)
	}
}

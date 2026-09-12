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
	"reflect"

	"github.com/jfsmig/onvif/v2/networking"
	"github.com/jfsmig/onvif/v2/xsd"
)

// anyURI is the type every URI a device reports arrives as.
var anyURI = reflect.TypeOf(xsd.AnyURI(""))

// redactURIs drops the account a device embedded in any URI reachable from v, which must be a
// pointer to the value to clean.
//
// It exists for the structs this package hands back whole. `onvif-cli dump` JSON-encodes a
// camera's reply wholesale, and a Capabilities block alone carries thirteen XAddr fields
// nested six types deep -- none of which passes through networking.AddEndpoint, because that
// only ever sees a copy on the way into the routing table. So a dump could show the
// credential stripped from the GetServices() map and preserved in the Capabilities printed
// just below it.
//
// Reflective rather than thirteen assignments, which is a deliberate trade. The finding
// behind this was that one family of URI had been missed while two others were handled by
// hand, so an enumeration of today's fields is the same mistake written down: a capability
// struct added to xsd/onvif later would reintroduce the leak silently. The cost is a walk
// over one reply per dump, which is not a hot path -- the per-call path is
// networking.AddEndpoint, and it is untouched.
//
// Only xsd.AnyURI is rewritten. Every URI in the ONVIF schema has that type, and nothing else
// does, so the walk cannot touch a field that merely looks like one.
func redactURIs(v any) {
	redactValue(reflect.ValueOf(v))
}

func redactValue(v reflect.Value) {
	// Anything reached through an unexported field is read-only: writing to it panics, and
	// for a map the panic comes from SetMapIndex rather than from a CanSet check further
	// down. Nothing behind an unexported field can reach a dump anyway, so the whole subtree
	// is skipped here rather than guarded in three places below.
	if !v.CanInterface() {
		return
	}

	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			redactValue(v.Elem())
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			redactValue(v.Index(i))
		}
	case reflect.Map:
		// A map value is not addressable, so it is cleaned as a copy and put back. Skipping
		// maps would be a hole exactly where a future reply keyed by token would fall.
		for _, key := range v.MapKeys() {
			item := reflect.New(v.Type().Elem()).Elem()
			item.Set(v.MapIndex(key))
			redactValue(item)
			v.SetMapIndex(key, item)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			field := v.Field(i)
			if field.Type() == anyURI {
				// CanSet is false for an unexported field, which no ONVIF reply has and
				// which could not be read out of a dump anyway.
				if field.CanSet() {
					field.SetString(networking.WithoutUserinfo(field.String()))
				}
				continue
			}
			redactValue(field)
		}
	}
}

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

// `subscribe` prints JSON Lines and not the whitespace-separated columns `discover` and
// `streams` print, and the reason is in this file. An event payload is nested key/value item
// lists (ONVIF Core section 9.4.1), so no fixed column count can hold it, and a SimpleItem
// Value may contain spaces -- which would break exactly the constant-field-count property
// that uuidColumn exists to preserve.
//
// Like interfaces_test.go and signal_test.go, this pins operator-facing properties of the
// tool rather than a wire format: the shape of a record, and the one-object-per-line contract
// every consumer of this output relies on.

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jfsmig/onvif/v2/networking"
	"github.com/jfsmig/onvif/v2/sdk"
)

// A fixed instant, so the rendering is a golden line rather than a clock read.
var recordedAt = time.Date(2026, 9, 10, 14, 2, 32, 118000000, time.UTC)

// crossing is the notification of ONVIF Core section 9.10.6, as sdk projects it.
var crossing = sdk.Notification{
	Topic:   "tns1:RuleEngine/LineDetector/Crossed",
	UtcTime: "2008-10-10T12:24:57.321Z",
	Source: []sdk.Item{
		{Name: "VideoSourceConfigurationToken", Value: "1"},
		{Name: "VideoAnalyticsConfigurationToken", Value: "2"},
		{Name: "Rule", Value: "MyImportantFence1"},
	},
	Data: []sdk.Item{{Name: "ObjectId", Value: "15"}},
}

func encodeRecords(t *testing.T, records ...eventRecord) string {
	t.Helper()

	var buf bytes.Buffer
	writer := &recordWriter{encoder: json.NewEncoder(&buf), cancel: func() {}}
	for _, record := range records {
		if err := writer.write(record); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return buf.String()
}

// The golden line: the key names, their order, the values as strings, and the alphabetical
// order encoding/json gives the maps. It is spelled out in full because every part of it is
// something an operator types into a jq filter, so a change to any of it breaks a command
// line somebody has saved.
func TestEventRecordShape(t *testing.T) {
	const want = `{"time":"2008-10-10T12:24:57.321Z","received":"2026-09-10T14:02:32.118Z",` +
		`"xaddr":"192.168.1.70:80",` +
		`"uuid":"urn:uuid:00000700-0013-0008-0203-ec71db76e907",` +
		`"topic":"tns1:RuleEngine/LineDetector/Crossed",` +
		`"source":{"Rule":"MyImportantFence1","VideoAnalyticsConfigurationToken":"2",` +
		`"VideoSourceConfigurationToken":"1"},` +
		`"data":{"ObjectId":"15"}}` + "\n"

	dev := networking.ClientInfo{
		Xaddr: "192.168.1.70:80",
		Uuid:  "urn:uuid:00000700-0013-0008-0203-ec71db76e907",
	}
	if got := encodeRecords(t, newEventRecord(recordedAt, dev, crossing)); got != want {
		t.Errorf("record =\n%s\nwant\n%s", got, want)
	}
}

// uuidColumn's "-" is documented as a rendering that keeps a constant field count in a
// whitespace-separated line, and as something that "must never travel into a credential
// lookup, where it would name a camera called '-'". JSON has no column to pad, so the key is
// absent instead -- which is also what makes has("uuid") and `.uuid // "unknown"` work.
func TestACameraNamedByAddressHasNoUuidKey(t *testing.T) {
	dev := networking.ClientInfo{Xaddr: "192.168.1.71:8000"}
	got := encodeRecords(t, newEventRecord(recordedAt, dev, crossing))

	if strings.Contains(got, `"uuid"`) {
		t.Errorf("a camera that reported no identifier still has a uuid key: %s", got)
	}
	if strings.Contains(got, `"-"`) {
		t.Errorf("the uuidColumn placeholder reached the JSON: %s", got)
	}
}

// The one-object-per-line contract. dumpSomething sets an indent on its encoder, and a
// copy-paste of that one line here would turn every record into ten and every downstream
// `while read` into nonsense.
func TestOneCompactObjectPerLine(t *testing.T) {
	// A value with spaces in it, which is the case that rules out columns: ONVIF puts no
	// constraint on a SimpleItem Value, and a rule named by a person has spaces in it.
	spaced := crossing
	spaced.Source = []sdk.Item{{Name: "Rule", Value: "My Important Fence 1"}}
	// And one large enough that a torn write would actually tear rather than happen to fit
	// inside a pipe's atomic write size.
	large := crossing
	large.Data = []sdk.Item{{Name: "Blob", Value: strings.Repeat("x", 8<<10)}}

	dev := networking.ClientInfo{Xaddr: "192.168.1.70:80"}
	got := encodeRecords(t,
		newEventRecord(recordedAt, dev, crossing),
		newEventRecord(recordedAt, dev, spaced),
		newEventRecord(recordedAt, dev, large))

	if n := strings.Count(got, "\n"); n != 3 {
		t.Errorf("3 records produced %d lines; json.Encoder without SetIndent emits exactly "+
			"one per Encode", n)
	}
	for i, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			t.Errorf("line %d is indented: %q", i, line)
		}
		var into map[string]any
		if err := json.Unmarshal([]byte(line), &into); err != nil {
			t.Errorf("line %d does not parse on its own: %v", i, err)
		}
	}
	if !strings.Contains(got, `"My Important Fence 1"`) {
		t.Errorf("a value containing spaces did not survive: %s", got)
	}
}

// Every value is a string, whatever the device meant by it. tt:SimpleItem/@Value is
// xs:anySimpleType, so one camera's ObjectId is a number and the next one's is a token, and a
// stream where .data.ObjectId is 15 on some lines and "15" on others cannot be filtered by
// one expression.
func TestEveryValueIsAString(t *testing.T) {
	typed := crossing
	typed.Data = []sdk.Item{
		{Name: "ObjectId", Value: "15"},
		{Name: "State", Value: "true"},
		{Name: "Likelihood", Value: "0.5"},
	}

	line := encodeRecords(t, newEventRecord(recordedAt,
		networking.ClientInfo{Xaddr: "192.168.1.70:80"}, typed))

	var into struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(line), &into); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for name, value := range into.Data {
		if _, ok := value.(string); !ok {
			t.Errorf("data.%s came back as %T, want a string — a per-camera guess makes one "+
				"jq filter work on one camera and fail on the next", name, value)
		}
	}
}

// A device may pretty-print the values it sends -- ONVIF's own examples in sections 9.10.4
// and 9.10.6 put the topic on its own indented line -- and a newline inside a record is the
// one thing a JSON-Lines stream must not contain. sdk trims on the way out of the projection;
// this is the property that depends on it.
func TestNoRecordContainsANewline(t *testing.T) {
	line := encodeRecords(t, newEventRecord(recordedAt,
		networking.ClientInfo{Xaddr: "192.168.1.70:80"}, crossing))

	if n := strings.Count(line, "\n"); n != 1 {
		t.Errorf("one record spans %d lines", n)
	}
	if strings.Contains(line, `\n`) {
		t.Errorf("a record carries an escaped newline, so a value was not trimmed: %s", line)
	}
}

// A stamp the device sent in another spelling of UTC is normalised, so that a stream from two
// cameras sorts and diffs as text. One that will not parse is passed through rather than
// dropped or replaced: it is the device's own answer, and hiding it would hide a defect.
func TestEventTime(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"the form ONVIF Core section 9.4.1 shows", "2008-10-10T12:24:57.321Z", "2008-10-10T12:24:57.321Z"},
		{"an offset becomes Z, so two cameras compare", "2026-09-10T16:02:31+02:00", "2026-09-10T14:02:31Z"},
		{"whole seconds stay whole seconds", "2026-09-10T14:02:31Z", "2026-09-10T14:02:31Z"},
		{"what will not parse is handed on as it stands", "not a time", "not a time"},
		{"and nothing stays nothing", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := eventTime(tc.in); got != tc.want {
				t.Errorf("eventTime(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// Section 9.4.1 requires item names to be unique within any group of a message, which is what
// makes a map safe. A device that breaks the rule keeps its last item rather than being
// rejected -- but the group is still there, which is the part worth asserting.
func TestADuplicateItemNameKeepsTheLast(t *testing.T) {
	got := itemMap([]sdk.Item{{Name: "Rule", Value: "first"}, {Name: "Rule", Value: "last"}})

	if len(got) != 1 || got["Rule"] != "last" {
		t.Errorf("itemMap = %v, want the last value of a duplicated name", got)
	}
}

// An empty group is absent rather than an empty object, so a filter can tell "the device sent
// no such list" from "it sent an empty one".
func TestAnEmptyGroupIsOmitted(t *testing.T) {
	line := encodeRecords(t, newEventRecord(recordedAt,
		networking.ClientInfo{Xaddr: "192.168.1.70:80"},
		sdk.Notification{Topic: "tns1:Device/HardwareFailure/FanFailure"}))

	for _, absent := range []string{`"source"`, `"key"`, `"data"`, `"operation"`} {
		if strings.Contains(line, absent) {
			t.Errorf("an event with no payload still carries %s: %s", absent, line)
		}
	}
}

// A stdout that cannot be written to ends the run: the stream is truncated, so there is
// nothing to be gained by pulling more of it, and every camera has to stop rather than pull
// into a closed file.
func TestAFailedWriteEndsTheRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	writer := &recordWriter{encoder: json.NewEncoder(brokenWriter{}), cancel: cancel}
	record := newEventRecord(recordedAt, networking.ClientInfo{Xaddr: "192.168.1.70:80"}, crossing)

	if err := writer.write(record); err == nil {
		t.Fatal("write reported success on a writer that cannot accept anything")
	}
	if ctx.Err() == nil {
		t.Error("the run was not stopped, so every camera keeps pulling into a closed file")
	}
	// The first failure is the one reported, and a second write does not retry.
	if writer.write(record) == nil {
		t.Error("a write after the failure reported success")
	}
	if writer.written() != 0 {
		t.Errorf("written() = %d after two failures", writer.written())
	}
	if writer.failure() == nil {
		t.Error("failure() is nil, so the run would exit 0 on a truncated stream")
	}
}

// brokenWriter stands in for a full disk under `> events.jsonl`.
type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errNoSpace }

var errNoSpace = &noSpaceError{}

type noSpaceError struct{}

func (*noSpaceError) Error() string { return "no space left on device" }

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
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
)

// eventRecord is one line of `subscribe` output: one ONVIF notification, flattened.
//
// The keys are lower case and they are ours; the names inside source, key and data are the
// device's own and are copied verbatim, because they come from the message description the
// camera publishes through GetEventProperties (ONVIF Core section 9.4.1) -- re-casing them
// would break the match with the very document that describes them. `dump` prints Go field
// names because it encodes sdk structs wholesale; this record is designed, and it is the one
// output of this tool whose field names an operator types by hand, in a jq filter.
//
// Every value is a string, on purpose. tt:SimpleItem/@Value is xs:anySimpleType, so one
// camera's ObjectId is a number and the next one's is a token, and a stream where
// .data.ObjectId is 15 on some lines and "15" on others cannot be filtered by one
// expression. It is also what lets an ElementItem, whose value is a fragment of XML, sit in
// the same map as a SimpleItem rather than needing a key of its own.
//
// Field order is JSON order -- encoding/json emits struct fields as declared -- so what
// identifies the line comes first and the payload last, which makes a screenful readable
// without jq at all. Inside the three maps the order is alphabetical, because encoding/json
// sorts map keys; an ItemList's order carries no meaning, since the name is the identity,
// and a stable order is what makes two lines diffable and this record's test a golden line.
type eventRecord struct {
	Time      string            `json:"time"`
	Received  string            `json:"received"`
	Xaddr     string            `json:"xaddr"`
	Uuid      string            `json:"uuid,omitempty"`
	Topic     string            `json:"topic"`
	Operation string            `json:"operation,omitempty"`
	Source    map[string]string `json:"source,omitempty"`
	Key       map[string]string `json:"key,omitempty"`
	Data      map[string]string `json:"data,omitempty"`
}

// newEventRecord shapes one notification from one camera.
//
// The instant is an argument rather than a call to time.Now, so the whole rendering is a
// golden line in a test -- the same reason deviceClockOffset takes one.
//
// uuid is absent when the camera reported none, and never the "-" that uuidColumn prints.
// That placeholder is documented where it lives as a rendering and nothing else, there to
// keep a constant field count in a whitespace-separated line; JSON has no column to pad, so
// the placeholder has no job here and would only reintroduce the confusion of a camera
// called "-". An absent key is also better than a null one: has("uuid"), .uuid // "unknown"
// and jq -e '.uuid' then all do the obvious thing.
//
// Both identities appear rather than one. xaddr is always available and is what an operator
// greps in a firewall log; uuid is the identity ONVIF Core section 7.3.1 requires to be
// constant across a device's network interfaces, and is the key a credentials file is named
// by -- so a camera whose lease moved keeps the second and changes the first.
func newEventRecord(now time.Time, dev networking.ClientInfo, notification sdk.Notification) eventRecord {
	return eventRecord{
		Time:      eventTime(notification.UtcTime),
		Received:  now.UTC().Format(time.RFC3339Nano),
		Xaddr:     dev.Xaddr,
		Uuid:      dev.Uuid,
		Topic:     notification.Topic,
		Operation: notification.Operation,
		Source:    itemMap(notification.Source),
		Key:       itemMap(notification.Key),
		Data:      itemMap(notification.Data),
	}
}

// eventTime normalises the stamp the device put on the message (ONVIF Core section 9.4.1),
// the only one that says when the event itself happened.
//
// Normalised rather than passed through because a stream is sorted, compared and diffed as
// text, and two cameras spelling the same instant "...Z" and "...+00:00" would not compare
// equal. RFC3339Nano trims trailing zeros, so a device reporting whole seconds still yields
// a whole-second stamp.
//
// The clock offset the client learnt for this device is deliberately not applied: never
// rewrite what the device said. A stamp that will not parse is passed through unchanged for
// the same reason -- it is the device's own answer, and a record that dropped it would hide
// a firmware defect instead of showing it.
func eventTime(utc string) string {
	if utc == "" {
		return ""
	}
	stamp, err := time.Parse(time.RFC3339, utc)
	if err != nil {
		return utc
	}
	return stamp.UTC().Format(time.RFC3339Nano)
}

// itemMap flattens one message group onto name -> value, and returns nil for an empty one so
// the key is omitted entirely: absence then means "the device sent no such list" rather than
// "it sent an empty one".
//
// A map is safe because ONVIF Core section 9.4.1 requires it: "The name of all Items shall
// be unique within all Items contained in any group of this Message." A device that breaks
// that rule has its last item win, and says so at trace level -- worth one line, because a
// silently merged pair would make a filter match nothing for no visible reason.
func itemMap(items []sdk.Item) map[string]string {
	if len(items) == 0 {
		return nil
	}

	out := make(map[string]string, len(items))
	for _, item := range items {
		if _, clash := out[item.Name]; clash {
			Logger.Trace().Str("item", item.Name).
				Msg("duplicate item name in one message group, keeping the last")
		}
		out[item.Name] = item.Value
	}
	return out
}

// recordWriter owns the only thing in this command that writes to stdout.
//
// The encoder is private and the lock is held across the whole Encode. A json.Encoder is not
// safe for concurrent use, so a shared one would be a data race that `go test -race` catches
// -- and a record carrying an ElementItem's XML can exceed a pipe's atomic write size, so a
// torn write is not theoretical. os.File.Write loops internally on a short write, so holding
// the lock across Encode is what stops another camera splicing itself into the middle of a
// line.
//
// A mutex and not a writer goroutine reading a channel: the back-pressure is the same, since
// a slow stdout blocks the camera that is writing and therefore stops it pulling, while a
// goroutine would add a bounded queue, a close-then-wait handshake and a way to lose the last
// records of a run at shutdown, for nothing.
//
// No SetIndent, unlike dumpSomething: a json.Encoder without one emits exactly one line per
// Encode, which is the whole of this output format. And no bufio, because `subscribe | jq` is
// watched live and a buffer that fills in an hour would show nothing for an hour.
type recordWriter struct {
	mu      sync.Mutex
	encoder *json.Encoder
	// cancel ends the run when stdout can no longer be written to, rather than leaving every
	// camera pulling into a closed file.
	cancel context.CancelFunc
	err    error
	count  int
}

// write emits one record, or reports the failure that has already ended the run.
func (w *recordWriter) write(record eventRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return w.err
	}
	if err := w.encoder.Encode(record); err != nil {
		// A closed pipe does not arrive here: Go turns EPIPE on fd 1 into SIGPIPE with its
		// default disposition, and only Interrupt and SIGTERM are trapped, so
		// `subscribe | head -1` ends the way every other Unix filter does. What does arrive
		// here is a full disk under `> events.jsonl`, and that fails the run: the stream is
		// truncated, so there is nothing to be gained by pulling more of it.
		w.err = fmt.Errorf("writing to stdout: %w", err)
		w.cancel()
		return w.err
	}
	w.count++
	return nil
}

// failure returns the error that ended the run, nil when stdout took everything.
func (w *recordWriter) failure() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}

// written returns how many records reached stdout.
func (w *recordWriter) written() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.count
}

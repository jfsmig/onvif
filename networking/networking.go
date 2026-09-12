// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// Portions of this file derive from the goonvif project, now use-go/onvif,
// originally distributed under the MIT License, see LICENSE.MIT,
// Copyright (c) 2018 Yakovlev Dmitry, Zhorzh Palanjyan, Crazybber.
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

package networking

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/beevik/etree"
	"github.com/jfsmig/go-wsd/gosoap"
	"github.com/jfsmig/onvif/utils"
)

// refuseRedirect stops net/http from following a 3xx.
//
// The ONVIF credential travels in the SOAP body, not in a header. http.NewRequest gives a
// *bytes.Buffer body a GetBody, so a 307 or 308 makes net/http replay the whole envelope —
// WS-Security UsernameToken included — at whatever host the Location names. Do only strips
// Authorization and Cookie across hosts, so its own protection does not cover this.
//
// Returning ErrUseLastResponse hands the 3xx back to the caller instead of erroring, so
// ReadAndParse reports it as "http request error: 307 ..." and an operator can see that the
// device asked to be redirected. A redirect is not part of the ONVIF SOAP binding anyway.
func refuseRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// SendSoap sends a SOAP message using the supplied client.
//
// The caller owns httpClient: a client obtained from NewClient refuses redirects, but one
// built elsewhere follows them by default and will replay the credential-bearing body to
// the redirect target. Set CheckRedirect on any client passed here directly.
func SendSoap(ctx context.Context, httpClient *http.Client, endpoint, message string) (*http.Response, error) {
	// NewRequestWithContext, not NewRequest followed by req.WithContext: the latter returns
	// a copy, so discarding it left every request on context.Background() — the caller's
	// deadline and cancellation reached nothing, and a device that accepted the connection
	// then went silent blocked forever. The context bounds the whole exchange, the response
	// body read included.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(message))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	return httpClient.Do(req)
}

// MaxResponseBytes bounds one SOAP reply.
//
// io.ReadAll grows without limit, so a device that streams forever -- broken firmware or a
// hostile host answering a discovery probe -- could exhaust the caller's memory. The cap is
// deliberately generous: the largest legitimate ONVIF reply is not a configuration but a
// log, since GetSystemLog and GetSystemSupportInformation return device text wholesale, and
// a tight limit would truncate a real answer.
//
// Per reply, which is the number to keep in mind when changing anything that fans out: the
// peak a caller can reach is this times the number of replies it reads at once. `dump all`
// offers about a hundred operations against one camera and http.Client's MaxConnsPerHost --
// 4 in bin/onvif-cli -- is what actually bounds the concurrent reads, so raising that raises
// the memory ceiling with it.
const MaxResponseBytes = 32 << 20

// ReadAndParse consumes the reply body and unmarshals it into reply.
//
// tag names the ONVIF operation and appears in every error. It was accepted and discarded
// before, which left an operator with "expected element type <Envelope>" and no way to tell
// which of the SDK's ~75 calls produced it -- and sdk.Fetch* returns no error to inspect
// either, so that message was the only evidence.
//
// The context that used to be the first parameter is gone: the exchange is already bound by
// http.NewRequestWithContext in SendSoap, which covers this body read, so checking ctx here
// would have been theatre.
func ReadAndParse(httpReply *http.Response, reply interface{}, tag string) error {
	// The body is read whatever the status, which it was not before. A non-200 returned on
	// the status alone and never looked, so the fault explaining it was discarded unread;
	// see soapFault.
	//
	// One byte past the cap, so a reply that sits exactly on it is still accepted and only
	// a genuinely oversized one is refused.
	b, err := io.ReadAll(io.LimitReader(httpReply.Body, MaxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("%s: %w", tag, err)
	}
	if len(b) > MaxResponseBytes {
		return fmt.Errorf("%s: reply exceeds %d bytes", tag, MaxResponseBytes)
	}

	if fault, ok := parseSOAPFault(b); ok {
		if httpReply.StatusCode != http.StatusOK {
			// Both, because a faulted non-200 is both things and callers already match
			// ErrHTTP. fmt.Errorf wraps every %w, so errors.Is finds either one.
			return fmt.Errorf("%w: %w: %s: %s", utils.ErrHTTP, utils.ErrSOAPFault, tag, fault)
		}
		return fmt.Errorf("%w: %s: %s", utils.ErrSOAPFault, tag, fault)
	}

	if httpReply.StatusCode != http.StatusOK {
		// Keep the status: telling 401 (wrong credentials) from 500 (device fault)
		// otherwise needs a packet capture. ErrHTTP stays the first wrapped error so
		// errors.Is keeps matching it.
		return fmt.Errorf("%w: %s: %s", utils.ErrHTTP, tag, httpReply.Status)
	}

	if err := xml.Unmarshal(b, reply); err != nil {
		return fmt.Errorf("%s: %w", tag, err)
	}
	return nil
}

// soapFault is the SOAP 1.2 fault, reduced to the three parts that identify it.
//
// ONVIF Core section 5.11.2.1 makes the fault the only channel for an operation error, and
// section 5.11.2.2 Table 5 makes the Subcode the normative discriminator: ter:NotAuthorized,
// ter:InvalidArgVal, ter:ActionNotSupported, ter:WellFormed, ter:TagMismatch. Without it a
// rejected credential and an unimplemented operation are the same 400 to a caller, and
// sdk.Fetch* renders both as an empty field.
//
// None of the tags names a namespace, so each matches on local name alone. That is
// deliberate: the fault arrives under whatever prefix the device bound to the SOAP envelope
// namespace, and the Subcode value is itself a prefixed QName ("ter:NotAuthorized") whose
// prefix is equally the device's choice. The prefix is kept verbatim rather than resolved,
// because it is what the specification's own tables print and what an operator searches for.
type soapFault struct {
	Code struct {
		Value   string `xml:"Value"`
		Subcode struct {
			Value string `xml:"Value"`
		} `xml:"Subcode"`
	} `xml:"Code"`
	Reason struct {
		Text string `xml:"Text"`
	} `xml:"Reason"`
}

func (f soapFault) String() string {
	code := f.Code.Value
	if sub := f.Code.Subcode.Value; sub != "" {
		code += "/" + sub
	}
	if f.Reason.Text == "" {
		return code
	}
	return code + ": " + f.Reason.Text
}

// parseSOAPFault reports the fault an envelope carries, and whether it carries one at all.
//
// A reply that is not a fault is not an error here: the generated response structs have no
// field a Fault can bind to, so an unmarshal against them succeeds on a fault body and
// yields a zero response -- which is exactly how a fault answered with HTTP 200, common in
// firmware and not conformant, used to reach a caller as a successful empty reply.
func parseSOAPFault(b []byte) (soapFault, bool) {
	// Cheap first, because every reply of every call reaches this and some are large: the
	// 32 MiB cap above is sized for GetSystemLog, and unmarshalling all of it a second time
	// to discover it holds no fault is a cost paid on the successful path. The element's
	// local name is literally "Fault" whatever prefix the device bound, so a body without
	// those five bytes cannot be one. A false positive costs only the structured parse below.
	if !bytes.Contains(b, []byte("Fault")) {
		return soapFault{}, false
	}

	var envelope struct {
		Body struct {
			Fault soapFault `xml:"Fault"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(b, &envelope); err != nil {
		return soapFault{}, false
	}
	fault := envelope.Body.Fault
	return fault, fault.Code.Value != "" || fault.Reason.Text != ""
}

// addWSATo emits the wsa:To header ONVIF Core section 9.10.5 shows on a request addressed
// to a subscription manager, and nothing at all when the URI is empty.
//
// Built as an element rather than through gosoap's AddStringHeaderContent: a subscription
// URI carries a query string ("...Subscription?Idx=0"), a device is free to put an ampersand
// in it, and etree escapes the character data it serialises -- while a string fragment would
// have to be escaped here, which would add a second escaping rule to this package for
// nothing. gosoap has no AddTo, and AddHeaderContent is the seam it exports instead; this is
// the same technique AddAction uses for wsa:Action.
//
// The wsa prefix resolves because Xlmns declares it on the envelope root.
func addWSATo(soap *gosoap.SoapMessage, to string) {
	if to == "" {
		return
	}
	element := etree.NewElement("wsa:To")
	element.SetText(to)
	soap.AddHeaderContent(element)
}

func buildMethodSOAP(msg string) (*gosoap.SoapMessage, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(msg); err != nil {
		return nil, err
	}
	element := doc.Root()

	soap := gosoap.NewEmptySOAP()
	soap.AddBodyContent(element)

	return soap, nil
}

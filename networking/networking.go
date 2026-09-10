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
	if httpReply.StatusCode != http.StatusOK {
		// Keep the status: telling 401 (wrong credentials) from 500 (device fault)
		// otherwise needs a packet capture. ErrHTTP stays the first wrapped error so
		// errors.Is keeps matching it.
		return fmt.Errorf("%w: %s: %s", utils.ErrHTTP, tag, httpReply.Status)
	}

	// One byte past the cap, so a reply that sits exactly on it is still accepted and only
	// a genuinely oversized one is refused.
	b, err := io.ReadAll(io.LimitReader(httpReply.Body, MaxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("%s: %w", tag, err)
	}
	if len(b) > MaxResponseBytes {
		return fmt.Errorf("%s: reply exceeds %d bytes", tag, MaxResponseBytes)
	}
	if err := xml.Unmarshal(b, reply); err != nil {
		return fmt.Errorf("%s: %w", tag, err)
	}
	return nil
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

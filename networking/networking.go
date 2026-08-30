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
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(message))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.WithContext(ctx)
	return httpClient.Do(req)
}

func ReadAndParse(ctx context.Context, httpReply *http.Response, reply interface{}, tag string) error {
	if httpReply.StatusCode != http.StatusOK {
		// Keep the status: telling 401 (wrong credentials) from 500 (device fault)
		// otherwise needs a packet capture.
		return fmt.Errorf("%w: %s", utils.ErrHTTP, httpReply.Status)
	}
	if b, err := io.ReadAll(httpReply.Body); err != nil {
		return err
	} else {
		return xml.Unmarshal(b, reply)
	}
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

type callMethodParams struct {
	Endpoint string
	Username string
	Password string
	Method   interface{}
}

package networking

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"github.com/beevik/etree"
	"github.com/jfsmig/onvif/gosoap"
	"io"
	"net/http"
)

// SendSoap send soap message
func SendSoap(httpClient *http.Client, endpoint, message string) (*http.Response, error) {
	return httpClient.Post(endpoint, "application/soap+xml; charset=utf-8", bytes.NewBufferString(message))
}

func ReadAndParse(ctx context.Context, httpReply *http.Response, reply interface{}, tag string) error {
	if httpReply.StatusCode/100 != 2 {
		return fmt.Errorf("unexpected status %v instead of 2XX", httpReply.StatusCode)
	}
	if b, err := io.ReadAll(httpReply.Body); err != nil {
		return err
	} else {
		return xml.Unmarshal(b, reply)
	}
}

func buildMethodSOAP(msg string) (gosoap.SoapMessage, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(msg); err != nil {
		return "", err
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

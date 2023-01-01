package networking

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"

	"github.com/beevik/etree"
	"github.com/jfsmig/onvif/gosoap"
	"github.com/jfsmig/onvif/utils"
)

// SendSoap send soap message
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
		return utils.ErrHttp
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

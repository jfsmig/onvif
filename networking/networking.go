package networking

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"github.com/beevik/etree"
	"github.com/juju/errors"
	"github.com/jfsmig/onvif/gosoap"
	"io/ioutil"
	"net/http"
)

// SendSoap send soap message
func SendSoap(httpClient *http.Client, endpoint, message string) (*http.Response, error) {
	resp, err := httpClient.Post(endpoint, "application/soap+xml; charset=utf-8", bytes.NewBufferString(message))
	if err != nil {
		return resp, errors.Annotate(err, "Post")
	}

	return resp, nil
}

func ReadAndParse(ctx context.Context, httpReply *http.Response, reply interface{}, tag string) error {
	if httpReply.StatusCode/100 != 2 {
		return fmt.Errorf("unexpected status %v instead of 2XX", httpReply.StatusCode)
	}
	if b, err := ioutil.ReadAll(httpReply.Body); err != nil {
		return errors.Annotate(err, "read")
	} else {
		err = xml.Unmarshal(b, reply)
		return errors.Annotate(err, "decode")
	}
}

func buildMethodSOAP(msg string) (gosoap.SoapMessage, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(msg); err != nil {
		//log.Println("Got error")

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

// callMethodDo functions call a method, defined <method> struct with authentication data
func callMethodDo(client *http.Client, params callMethodParams) (*http.Response, error) {
	output, err := xml.MarshalIndent(params.Method, "  ", "    ")
	if err != nil {
		return nil, err
	}

	soap, err := buildMethodSOAP(string(output))
	if err != nil {
		return nil, err
	}

	soap.AddRootNamespaces(Xlmns)
	soap.AddAction()

	//Auth Handling
	if params.Username != "" && params.Password != "" {
		soap.AddWSSecurity(params.Username, params.Password)
	}

	return SendSoap(client, params.Endpoint, soap.String())
}

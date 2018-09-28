package transport

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func DefaultHTTPClient(insecure bool) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	c := &http.Client{Transport: transport, Timeout: time.Second * 10}
	return c
}

func ResponseCloser(response *http.Response) {
	if err := response.Body.Close(); err != nil {
		logrus.Error("failed to close response body")
	}
}

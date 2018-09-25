package transport

import (
	"crypto/tls"
	"net/http"
	"time"
)

func DefaultHTTPClient(insecure bool) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	c := &http.Client{Transport: transport, Timeout: time.Second * 10}
	return c
}

package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// dialContextFunc is a type for the DialContext function
type dialContextFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// createTransport creates an HTTP transport with the specified configuration
func createTransport(useHTTP1, useHTTP11, noKeepAlive bool, dialContext dialContextFunc) *http.Transport {
	switch {
	case useHTTP1:
		return &http.Transport{
			TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			TLSClientConfig: &tls.Config{
				MaxVersion: tls.VersionTLS12,
			},
			ForceAttemptHTTP2:     false,
			DisableKeepAlives:     noKeepAlive,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableCompression:    true,
			DialContext:           dialContext,
		}
	case useHTTP11:
		return &http.Transport{
			TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			ForceAttemptHTTP2:     false,
			DisableKeepAlives:     noKeepAlive,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableCompression:    true,
			DialContext:           dialContext,
		}
	default: // HTTP/2 is default
		return &http.Transport{
			ForceAttemptHTTP2:     true,
			DisableKeepAlives:     noKeepAlive,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableCompression:    true,
			DialContext:           dialContext,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: false,
				ClientSessionCache: tls.NewLRUClientSessionCache(100),
			},
		}
	}
}

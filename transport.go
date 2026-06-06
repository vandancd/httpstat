package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

type dialContextFunc func(ctx context.Context, network, addr string) (net.Conn, error)

func baseTransport(noKeepAlive bool, dialContext dialContextFunc) *http.Transport {
	return &http.Transport{
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
}

func createTransport(useHTTP1, useHTTP11, noKeepAlive bool, dialContext dialContextFunc) *http.Transport {
	switch {
	case useHTTP1:
		t := baseTransport(noKeepAlive, dialContext)
		t.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
		t.TLSClientConfig = &tls.Config{MaxVersion: tls.VersionTLS12}
		t.ForceAttemptHTTP2 = false
		return t
	case useHTTP11:
		t := baseTransport(noKeepAlive, dialContext)
		t.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
		t.ForceAttemptHTTP2 = false
		return t
	default:
		t := baseTransport(noKeepAlive, dialContext)
		t.ForceAttemptHTTP2 = true
		t.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
			ClientSessionCache: tls.NewLRUClientSessionCache(100),
		}
		return t
	}
}

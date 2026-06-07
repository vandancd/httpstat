package probe

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type customDialer struct {
	*net.Dialer
	preferIPv6 bool
}

func (d *customDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if d.preferIPv6 {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip6", host)
		if err != nil || len(ips) == 0 {
			return d.Dialer.DialContext(ctx, network, address)
		}
		for _, ip := range ips {
			if ip.To4() == nil {
				conn, err := d.Dialer.DialContext(ctx, "tcp6", net.JoinHostPort(ip.String(), port))
				if err == nil {
					return conn, nil
				}
			}
		}
	}
	return d.Dialer.DialContext(ctx, network, address)
}

func Run(ctx context.Context, url string, opts Options) (Result, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	log := &TraceLog{}
	useCustomDNS := len(opts.DNSServers) > 0

	var netResolver *net.Resolver
	if useCustomDNS {
		netResolver = createCustomResolver(opts.DNSServers, log)
	}

	dialer := &customDialer{
		Dialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			Resolver:  netResolver,
			DualStack: !opts.PreferIPv6,
		},
		preferIPv6: opts.PreferIPv6,
	}

	transport := createTransport(opts.UseHTTP1, opts.UseHTTP11, opts.NoKeepAlive, dialer.DialContext)
	m := NewMeasurement(log, useCustomDNS)
	redirects := make([]RedirectInfo, 0)

	rh := &RedirectHandler{
		log:          log,
		maxRedirects: opts.MaxRedirects,
		useCustomDNS: useCustomDNS,
		redirects:    &redirects,
	}

	client := &http.Client{
		Transport:     transport,
		Timeout:       opts.Timeout,
		CheckRedirect: rh.Handle,
	}

	req, err := createRequest(ctx, url, m, opts)
	if err != nil {
		return Result{}, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if err := processResponseBody(resp, m, time.Now()); err != nil {
		return Result{}, fmt.Errorf("reading response body: %w", err)
	}

	return Result{
		URL:           resp.Request.URL.String(),
		HTTPProtocol:  resp.Proto,
		StatusCode:    resp.StatusCode,
		Status:        resp.Status,
		Redirects:     redirects,
		Timing:        m.Result(),
		StartTime:     m.StartTime(),
		TraceMessages: log.Flush(),
	}, nil
}

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// Global variable to track if we're using a custom resolver
var resolver *net.Resolver

// customDialer extends net.Dialer with IPv6 preference
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

		// Resolve the IP addresses
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip6", host)
		if err != nil || len(ips) == 0 {
			// Fallback to original dialer if IPv6 is not available
			return d.Dialer.DialContext(ctx, network, address)
		}

		// Try IPv6 addresses first
		for _, ip := range ips {
			if ip.To4() == nil { // Ensure it's an IPv6 address
				ipv6Addr := net.JoinHostPort(ip.String(), port)
				conn, err := d.Dialer.DialContext(ctx, "tcp6", ipv6Addr)
				if err == nil {
					return conn, nil
				}
			}
		}
	}

	// Fallback to original dialer
	return d.Dialer.DialContext(ctx, network, address)
}

func main() {
	// Parse command line flags
	fs := flag.NewFlagSet("httpstat", flag.ContinueOnError)
	http1 := fs.Bool("http1", false, "Use HTTP/1.0")
	http11 := fs.Bool("http1.1", false, "Use HTTP/1.1")
	noKeepAlive := fs.Bool("no-keepalive", false, "Disable keep-alive connections")
	timeout := fs.Int("timeout", 60, "Timeout in seconds (default: 60)")
	maxRedirects := fs.Int("max-redirects", 5, "Maximum number of redirects allowed (default: 5, range: 2-10)")
	dnsServers := fs.String("dns-servers", "", "Comma-separated list of DNS server IP addresses (e.g., 8.8.8.8,8.8.4.4)")
	useIPv6 := fs.Bool("ipv6", false, "Prefer IPv6 connections over IPv4")

	// Parse command line arguments
	url, err := parseCommandLine(fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Validate max redirects
	if *maxRedirects < 2 || *maxRedirects > 10 {
		fmt.Fprintf(os.Stderr, "Error: max-redirects must be between 2 and 10\n")
		os.Exit(1)
	}

	// Set up DNS resolver if custom servers are provided
	if *dnsServers != "" {
		servers := strings.Split(*dnsServers, ",")
		for i, server := range servers {
			servers[i] = strings.TrimSpace(server)
		}
		resolver = createCustomResolver(servers)
	}

	// Create base dialer
	baseDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  resolver,
		DualStack: !*useIPv6, // Disable dual stack (Happy Eyeballs) when IPv6 is preferred
	}

	// Create custom dialer with IPv6 preference
	dialer := &customDialer{
		Dialer:     baseDialer,
		preferIPv6: *useIPv6,
	}

	// Create transport and initialize tracking variables
	transport := createTransport(*http1, *http11, *noKeepAlive, dialer.DialContext)
	url = normalizeURL(url)
	redirects := make([]RedirectInfo, 0)
	var finalTiming Timing

	// Create HTTP client
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(*timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return handleRedirect(req, via, &redirects, *maxRedirects)
		},
	}

	// Create and execute request
	req, err := createRequest(url, &finalTiming)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}

	// Execute request and process response
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Process response body and timing
	bodyStart := time.Now()
	if err := processResponseBody(resp, &finalTiming, bodyStart, start); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing response: %v\n", err)
		os.Exit(1)
	}

	// Print results
	printResults(resp, redirects, finalTiming)

	/*dnsTraceErr := traceDNS("www.vandan.com")
	if dnsTraceErr != nil {
		fmt.Fprintf(os.Stderr, "DNS trace failed: %v\n", err)
	}*/
}

func parseCommandLine(fs *flag.FlagSet) (string, error) {
	var url string
	var args []string
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			url = arg
		} else {
			args = append(args, arg)
		}
	}

	if err := fs.Parse(args); err != nil {
		return "", fmt.Errorf("error parsing flags: %v", err)
	}

	if url == "" {
		return "", fmt.Errorf("usage: %s [--http1 | --http1.1 | --http2] [--no-keepalive] [--timeout seconds] [--max-redirects count] [--dns-servers server1,server2] <url>", os.Args[0])
	}

	return url, nil
}

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	fs := flag.NewFlagSet("httpstat", flag.ContinueOnError)
	http1 := fs.Bool("http1", false, "Use HTTP/1.0")
	http11 := fs.Bool("http1.1", false, "Use HTTP/1.1")
	noKeepAlive := fs.Bool("no-keepalive", false, "Disable keep-alive connections")
	timeout := fs.Int("timeout", 60, "Timeout in seconds (default: 60)")
	maxRedirects := fs.Int("max-redirects", 5, "Maximum number of redirects allowed (default: 5, range: 2-10)")
	dnsServers := fs.String("dns-servers", "", "Comma-separated list of DNS server IP addresses (e.g., 8.8.8.8,8.8.4.4)")
	useIPv6 := fs.Bool("ipv6", false, "Prefer IPv6 connections over IPv4")
	browser := fs.Bool("browser", false, "Use headless browser probe")
	asJSON := fs.Bool("json", false, "Output results as JSON")
	showTrace := fs.Bool("trace", false, "Show trace messages in output")

	url, err := parseCommandLine(fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if *maxRedirects < 2 || *maxRedirects > 10 {
		fmt.Fprintf(os.Stderr, "Error: max-redirects must be between 2 and 10\n")
		os.Exit(1)
	}

	if *browser {
		if err := runBrowserProbe(url); err != nil {
			fmt.Fprintf(os.Stderr, "Browser probe failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	var servers []string
	if *dnsServers != "" {
		for _, s := range strings.Split(*dnsServers, ",") {
			servers = append(servers, strings.TrimSpace(s))
		}
	}

	result, err := Run(url, ProbeOptions{
		UseHTTP1:     *http1,
		UseHTTP11:    *http11,
		NoKeepAlive:  *noKeepAlive,
		Timeout:      time.Duration(*timeout) * time.Second,
		MaxRedirects: *maxRedirects,
		DNSServers:   servers,
		PreferIPv6:   *useIPv6,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		printJSON(result)
	} else {
		printWaterfall(result, *showTrace)
	}
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

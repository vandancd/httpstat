package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	probe "github.com/vandancd/httpstat/pkg/probe"
)

const version = "1.2.0"

type headerFlags []string

func (h *headerFlags) String() string { return strings.Join(*h, ", ") }
func (h *headerFlags) Set(v string) error {
	if !strings.Contains(v, ":") {
		return fmt.Errorf("invalid header %q: must be in \"Key: Value\" format", v)
	}
	*h = append(*h, v)
	return nil
}

func validateAndBuildOptions(
	userAgent string,
	userAgentExplicit bool,
	http1, http11, noKeepAlive bool,
	timeout int,
	maxRedirects int,
	dnsServers []string,
	useIPv6 bool,
	method string,
	headers []string,
	bearer string,
	body string,
	jsonBody string,
) (probe.Options, error) {
	if userAgentExplicit {
		for _, h := range headers {
			name := strings.TrimSpace(strings.SplitN(h, ":", 2)[0])
			if strings.EqualFold(name, "user-agent") {
				return probe.Options{}, fmt.Errorf("conflict: --user-agent and -H \"User-Agent: ...\" cannot both be set")
			}
		}
	}
	if bearer != "" {
		for _, h := range headers {
			name := strings.TrimSpace(strings.SplitN(h, ":", 2)[0])
			if strings.EqualFold(name, "authorization") {
				return probe.Options{}, fmt.Errorf("conflict: --bearer and -H \"Authorization: ...\" cannot both be set")
			}
		}
	}
	if body != "" && jsonBody != "" {
		return probe.Options{}, fmt.Errorf("conflict: --data and --json cannot both be set")
	}
	if maxRedirects < 2 || maxRedirects > 10 {
		return probe.Options{}, fmt.Errorf("max-redirects must be between 2 and 10 (got %d)", maxRedirects)
	}
	effectiveMethod := method
	if effectiveMethod == "" && (body != "" || jsonBody != "") {
		effectiveMethod = "POST"
	}
	return probe.Options{
		UseHTTP1:    http1,
		UseHTTP11:   http11,
		NoKeepAlive: noKeepAlive,
		Timeout:     time.Duration(timeout) * time.Second,
		MaxRedirects: maxRedirects,
		DNSServers:  dnsServers,
		PreferIPv6:  useIPv6,
		UserAgent:   userAgent,
		Method:      effectiveMethod,
		Headers:     headers,
		BearerToken: bearer,
		Body:        body,
		JSONBody:    jsonBody,
	}, nil
}

func main() {
	fs := flag.NewFlagSet("httpstat", flag.ContinueOnError)
	showVersion := fs.Bool("version", false, "Print version and exit")
	userAgent := fs.String("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36", "User-Agent header to send")
	http1 := fs.Bool("http1", false, "Use HTTP/1.0")
	http11 := fs.Bool("http1.1", false, "Use HTTP/1.1")
	noKeepAlive := fs.Bool("no-keepalive", false, "Disable keep-alive connections")
	timeout := fs.Int("timeout", 60, "Timeout in seconds (default: 60)")
	maxRedirects := fs.Int("max-redirects", 5, "Maximum number of redirects allowed (default: 5, range: 2-10)")
	dnsServers := fs.String("dns-servers", "", "Comma-separated list of DNS server IP addresses (e.g., 8.8.8.8,8.8.4.4)")
	useIPv6 := fs.Bool("ipv6", false, "Prefer IPv6 connections over IPv4")
	asJSON := fs.Bool("output-json", false, "Output results as JSON")
	showTrace := fs.Bool("trace", false, "Show trace messages in output")

	var method string
	fs.StringVar(&method, "X", "", "HTTP method (e.g. GET, POST, PUT, DELETE)")
	fs.StringVar(&method, "method", "", "HTTP method (alias for -X)")

	var headers headerFlags
	fs.Var(&headers, "H", "Request header in \"Key: Value\" format (repeatable)")
	fs.Var(&headers, "header", "Request header in \"Key: Value\" format (alias for -H)")

	bearer := fs.String("bearer", "", "Bearer token for Authorization header")

	var body string
	fs.StringVar(&body, "d", "", "Request body (raw string literal)")
	fs.StringVar(&body, "data", "", "Request body (alias for -d)")

	jsonBody := fs.String("json", "", "Request body as JSON; sets Content-Type and Accept to application/json")

	url, err := parseCommandLine(fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if *showVersion {
		fmt.Printf("httpstat %s\n", version)
		return
	}

	if url == "" {
		fmt.Fprintf(os.Stderr, "usage: %s [--http1 | --http1.1 | --http2] [--no-keepalive] [--timeout seconds] [--max-redirects count] [--dns-servers server1,server2] <url>\n", os.Args[0])
		os.Exit(1)
	}

	var servers []string
	if *dnsServers != "" {
		for _, s := range strings.Split(*dnsServers, ",") {
			servers = append(servers, strings.TrimSpace(s))
		}
	}

	var userAgentExplicit bool
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "user-agent" {
			userAgentExplicit = true
		}
	})

	opts, err := validateAndBuildOptions(
		*userAgent, userAgentExplicit,
		*http1, *http11, *noKeepAlive,
		*timeout, *maxRedirects,
		servers, *useIPv6,
		method, []string(headers), *bearer, body, *jsonBody,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result, err := probe.Run(context.Background(), url, opts)
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
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		return "", fmt.Errorf("error parsing flags: %v", err)
	}
	if args := fs.Args(); len(args) > 0 {
		return args[0], nil
	}
	return "", nil
}

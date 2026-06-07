package main

import (
	"strings"
	"testing"
)

// helpers to build a minimal valid call to validateAndBuildOptions.
// All parameters that are not under test get safe defaults.
func defaultOpts() (
	userAgent string,
	userAgentExplicit bool,
	http1, http11, noKeepAlive bool,
	timeout, maxRedirects int,
	dnsServers []string,
	useIPv6 bool,
	method string,
	headers []string,
	bearer, body, jsonBody string,
) {
	return "TestAgent", false,
		false, false, false,
		30, 5,
		nil,
		false,
		"", nil,
		"", "", ""
}

// TestValidate_UserAgentConflict ensures that supplying --user-agent explicitly
// AND a -H "User-Agent: ..." header returns an error.
func TestValidate_UserAgentConflict(t *testing.T) {
	ua, _, h1, h11, nka, to, mr, dns, v6, meth, _, br, bd, jb := defaultOpts()
	_, err := validateAndBuildOptions(
		ua, true, // userAgentExplicit = true
		h1, h11, nka, to, mr, dns, v6,
		meth,
		[]string{"User-Agent: CustomBot/1.0"},
		br, bd, jb,
	)
	if err == nil {
		t.Fatal("expected error for --user-agent + -H User-Agent conflict, got nil")
	}
	if !strings.Contains(err.Error(), "User-Agent") {
		t.Errorf("error should mention User-Agent, got: %v", err)
	}
}

// TestValidate_UserAgentNoConflict_NotExplicit: if --user-agent was NOT
// explicitly set, providing a -H "User-Agent: ..." header is fine.
func TestValidate_UserAgentNoConflict_NotExplicit(t *testing.T) {
	ua, _, h1, h11, nka, to, mr, dns, v6, meth, _, br, bd, jb := defaultOpts()
	_, err := validateAndBuildOptions(
		ua, false, // userAgentExplicit = false
		h1, h11, nka, to, mr, dns, v6,
		meth,
		[]string{"User-Agent: CustomBot/1.0"},
		br, bd, jb,
	)
	if err != nil {
		t.Fatalf("unexpected error when --user-agent not explicit: %v", err)
	}
}

// TestValidate_BearerConflict ensures that --bearer and -H "Authorization: ..."
// together return an error.
func TestValidate_BearerConflict(t *testing.T) {
	ua, uae, h1, h11, nka, to, mr, dns, v6, meth, _, _, bd, jb := defaultOpts()
	_, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to, mr, dns, v6,
		meth,
		[]string{"Authorization: Basic dXNlcjpwYXNz"},
		"mytoken", // bearer set
		bd, jb,
	)
	if err == nil {
		t.Fatal("expected error for --bearer + -H Authorization conflict, got nil")
	}
	if !strings.Contains(err.Error(), "Authorization") {
		t.Errorf("error should mention Authorization, got: %v", err)
	}
}

// TestValidate_DataAndJSONConflict ensures --data and --json together error.
func TestValidate_DataAndJSONConflict(t *testing.T) {
	ua, uae, h1, h11, nka, to, mr, dns, v6, meth, hdrs, br, _, _ := defaultOpts()
	_, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to, mr, dns, v6,
		meth, hdrs, br,
		"raw body",       // body
		`{"key":"value"}`, // jsonBody
	)
	if err == nil {
		t.Fatal("expected error for --data + --json conflict, got nil")
	}
	if !strings.Contains(err.Error(), "--data") && !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --data or --json, got: %v", err)
	}
}

// TestValidate_MaxRedirects_TooLow: maxRedirects=1 should error.
func TestValidate_MaxRedirects_TooLow(t *testing.T) {
	ua, uae, h1, h11, nka, to, _, dns, v6, meth, hdrs, br, bd, jb := defaultOpts()
	_, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to,
		1, // maxRedirects
		dns, v6, meth, hdrs, br, bd, jb,
	)
	if err == nil {
		t.Fatal("expected error for maxRedirects=1, got nil")
	}
}

// TestValidate_MaxRedirects_TooHigh: maxRedirects=11 should error.
func TestValidate_MaxRedirects_TooHigh(t *testing.T) {
	ua, uae, h1, h11, nka, to, _, dns, v6, meth, hdrs, br, bd, jb := defaultOpts()
	_, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to,
		11, // maxRedirects
		dns, v6, meth, hdrs, br, bd, jb,
	)
	if err == nil {
		t.Fatal("expected error for maxRedirects=11, got nil")
	}
}

// TestValidate_MaxRedirects_MinValid: maxRedirects=2 should succeed.
func TestValidate_MaxRedirects_MinValid(t *testing.T) {
	ua, uae, h1, h11, nka, to, _, dns, v6, meth, hdrs, br, bd, jb := defaultOpts()
	_, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to,
		2, // maxRedirects
		dns, v6, meth, hdrs, br, bd, jb,
	)
	if err != nil {
		t.Fatalf("unexpected error for maxRedirects=2: %v", err)
	}
}

// TestValidate_MaxRedirects_MaxValid: maxRedirects=10 should succeed.
func TestValidate_MaxRedirects_MaxValid(t *testing.T) {
	ua, uae, h1, h11, nka, to, _, dns, v6, meth, hdrs, br, bd, jb := defaultOpts()
	_, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to,
		10, // maxRedirects
		dns, v6, meth, hdrs, br, bd, jb,
	)
	if err != nil {
		t.Fatalf("unexpected error for maxRedirects=10: %v", err)
	}
}

// TestValidate_ImplicitPOST_FromBody: no method + data body → method is POST.
func TestValidate_ImplicitPOST_FromBody(t *testing.T) {
	ua, uae, h1, h11, nka, to, mr, dns, v6, _, hdrs, br, _, jb := defaultOpts()
	opts, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to, mr, dns, v6,
		"",           // method empty
		hdrs, br,
		"hello=world", // body provided
		jb,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Method != "POST" {
		t.Errorf("expected Method=POST when body set and method empty, got %q", opts.Method)
	}
}

// TestValidate_ImplicitPOST_FromJSONBody: no method + json body → method is POST.
func TestValidate_ImplicitPOST_FromJSONBody(t *testing.T) {
	ua, uae, h1, h11, nka, to, mr, dns, v6, _, hdrs, br, bd, _ := defaultOpts()
	opts, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to, mr, dns, v6,
		"",  // method empty
		hdrs, br, bd,
		`{"x":1}`, // jsonBody provided
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Method != "POST" {
		t.Errorf("expected Method=POST when jsonBody set and method empty, got %q", opts.Method)
	}
}

// TestValidate_ExplicitMethodWins: explicit method + body → explicit method is kept.
func TestValidate_ExplicitMethodWins(t *testing.T) {
	ua, uae, h1, h11, nka, to, mr, dns, v6, _, hdrs, br, _, jb := defaultOpts()
	opts, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to, mr, dns, v6,
		"PUT",         // explicit method
		hdrs, br,
		"some data",   // body
		jb,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Method != "PUT" {
		t.Errorf("expected Method=PUT (explicit), got %q", opts.Method)
	}
}

// TestValidate_NoBodyNoMethod: no body + no method → Method is empty string
// (probe defaults to GET when creating the request).
func TestValidate_NoBodyNoMethod(t *testing.T) {
	ua, uae, h1, h11, nka, to, mr, dns, v6, _, hdrs, br, _, _ := defaultOpts()
	opts, err := validateAndBuildOptions(
		ua, uae, h1, h11, nka, to, mr, dns, v6,
		"", // method empty
		hdrs, br,
		"", // body empty
		"", // jsonBody empty
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Method != "" {
		t.Errorf("expected Method=\"\" when no body and no method, got %q", opts.Method)
	}
}

// TestValidate_ValidInputs_OptionsPopulated: sanity-check that returned Options
// contain the supplied values when everything is valid.
func TestValidate_ValidInputs_OptionsPopulated(t *testing.T) {
	opts, err := validateAndBuildOptions(
		"MyAgent", false,
		true, false, true,
		45, 7,
		[]string{"1.1.1.1"},
		true,
		"DELETE",
		[]string{"X-Custom: value"},
		"", "", "",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.UserAgent != "MyAgent" {
		t.Errorf("UserAgent: got %q want %q", opts.UserAgent, "MyAgent")
	}
	if !opts.UseHTTP1 {
		t.Error("UseHTTP1 should be true")
	}
	if !opts.NoKeepAlive {
		t.Error("NoKeepAlive should be true")
	}
	if opts.MaxRedirects != 7 {
		t.Errorf("MaxRedirects: got %d want 7", opts.MaxRedirects)
	}
	if len(opts.DNSServers) != 1 || opts.DNSServers[0] != "1.1.1.1" {
		t.Errorf("DNSServers: got %v", opts.DNSServers)
	}
	if !opts.PreferIPv6 {
		t.Error("PreferIPv6 should be true")
	}
	if opts.Method != "DELETE" {
		t.Errorf("Method: got %q want DELETE", opts.Method)
	}
	if len(opts.Headers) != 1 || opts.Headers[0] != "X-Custom: value" {
		t.Errorf("Headers: got %v", opts.Headers)
	}
}

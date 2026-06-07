package probe

import (
	"testing"
)

func TestBuildHeaders_Empty(t *testing.T) {
	h := buildHeaders(Options{})
	if len(h) != 0 {
		t.Errorf("expected empty header map, got %v", h)
	}
}

func TestBuildHeaders_JSONBody(t *testing.T) {
	h := buildHeaders(Options{JSONBody: `{"key":"value"}`})
	if got := h.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", got, "application/json")
	}
	if got := h.Get("Accept"); got != "application/json" {
		t.Errorf("Accept: got %q, want %q", got, "application/json")
	}
}

func TestBuildHeaders_UserAgent(t *testing.T) {
	h := buildHeaders(Options{UserAgent: "TestAgent/1.0"})
	if got := h.Get("User-Agent"); got != "TestAgent/1.0" {
		t.Errorf("User-Agent: got %q, want %q", got, "TestAgent/1.0")
	}
}

func TestBuildHeaders_BearerToken(t *testing.T) {
	h := buildHeaders(Options{BearerToken: "mytoken123"})
	if got := h.Get("Authorization"); got != "Bearer mytoken123" {
		t.Errorf("Authorization: got %q, want %q", got, "Bearer mytoken123")
	}
}

func TestBuildHeaders_ExplicitHeadersOverrideJSON(t *testing.T) {
	h := buildHeaders(Options{
		JSONBody: `{"a":1}`,
		Headers:  []string{"Content-Type: text/plain"},
	})
	if got := h.Get("Content-Type"); got != "text/plain" {
		t.Errorf("Content-Type: got %q, want %q", got, "text/plain")
	}
	if got := h.Get("Accept"); got != "application/json" {
		t.Errorf("Accept: got %q, want %q", got, "application/json")
	}
}

func TestBuildHeaders_MultipleExplicitHeaders(t *testing.T) {
	h := buildHeaders(Options{
		Headers: []string{
			"X-Custom-One: alpha",
			"X-Custom-Two: beta",
		},
	})
	if got := h.Get("X-Custom-One"); got != "alpha" {
		t.Errorf("X-Custom-One: got %q, want %q", got, "alpha")
	}
	if got := h.Get("X-Custom-Two"); got != "beta" {
		t.Errorf("X-Custom-Two: got %q, want %q", got, "beta")
	}
}

func TestBuildHeaders_ExplicitHeadersOverrideBearer(t *testing.T) {
	h := buildHeaders(Options{
		BearerToken: "ignored",
		Headers:     []string{"Authorization: Basic dXNlcjpwYXNz"},
	})
	if got := h.Get("Authorization"); got != "Basic dXNlcjpwYXNz" {
		t.Errorf("Authorization: got %q, want %q", got, "Basic dXNlcjpwYXNz")
	}
}

func TestBuildHeaders_NoUserAgentWhenEmpty(t *testing.T) {
	h := buildHeaders(Options{UserAgent: ""})
	if got := h.Get("User-Agent"); got != "" {
		t.Errorf("expected no User-Agent, got %q", got)
	}
}

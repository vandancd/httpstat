package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// renderBar
// ---------------------------------------------------------------------------

func TestRenderBar_ZeroDuration(t *testing.T) {
	out := renderBar(0, 100*time.Millisecond, colorCyan)
	if strings.Contains(out, "█") {
		t.Errorf("expected no filled blocks for d=0, got %q", out)
	}
}

func TestRenderBar_FullDuration(t *testing.T) {
	d := 100 * time.Millisecond
	out := renderBar(d, d, colorCyan)
	blockCount := strings.Count(out, "█")
	if blockCount != barWidth {
		t.Errorf("expected %d filled blocks for d==grandTotal, got %d", barWidth, blockCount)
	}
}

func TestRenderBar_ZeroGrandTotal(t *testing.T) {
	out := renderBar(50*time.Millisecond, 0, colorCyan)
	if strings.Contains(out, "█") {
		t.Errorf("expected no filled blocks when grandTotal=0, got %q", out)
	}
}

func TestRenderBar_MidValue(t *testing.T) {
	grandTotal := 100 * time.Millisecond
	d := 50 * time.Millisecond
	out := renderBar(d, grandTotal, colorCyan)
	blockCount := strings.Count(out, "█")
	expected := barWidth / 2
	if blockCount != expected {
		t.Errorf("expected ~%d filled blocks for d=grandTotal/2, got %d", expected, blockCount)
	}
}

func TestRenderBar_TotalLengthAlwaysBarWidth(t *testing.T) {
	cases := []struct {
		d, grandTotal time.Duration
	}{
		{0, 100 * time.Millisecond},
		{30 * time.Millisecond, 100 * time.Millisecond},
		{100 * time.Millisecond, 100 * time.Millisecond},
	}
	for _, c := range cases {
		out := renderBar(c.d, c.grandTotal, colorCyan)
		blocks := strings.Count(out, "█")
		fillers := strings.Count(out, "░")
		if blocks+fillers != barWidth {
			t.Errorf("d=%v grandTotal=%v: blocks(%d)+fillers(%d) != barWidth(%d)",
				c.d, c.grandTotal, blocks, fillers, barWidth)
		}
	}
}

// ---------------------------------------------------------------------------
// ttfbColor
// ---------------------------------------------------------------------------

func TestTTFBColor_Green(t *testing.T) {
	for _, d := range []time.Duration{0, 1 * time.Millisecond, 199 * time.Millisecond} {
		if got := ttfbColor(d); got != colorGreen {
			t.Errorf("ttfbColor(%v) = %q, want colorGreen", d, got)
		}
	}
}

func TestTTFBColor_YellowAt200ms(t *testing.T) {
	// 200ms is the boundary — must be yellow, not green
	if got := ttfbColor(200 * time.Millisecond); got != colorYellow {
		t.Errorf("ttfbColor(200ms) = %q, want colorYellow", got)
	}
}

func TestTTFBColor_Yellow(t *testing.T) {
	for _, d := range []time.Duration{201 * time.Millisecond, 499 * time.Millisecond} {
		if got := ttfbColor(d); got != colorYellow {
			t.Errorf("ttfbColor(%v) = %q, want colorYellow", d, got)
		}
	}
}

func TestTTFBColor_RedAt500ms(t *testing.T) {
	// 500ms is the boundary — must be red, not yellow
	if got := ttfbColor(500 * time.Millisecond); got != colorRed {
		t.Errorf("ttfbColor(500ms) = %q, want colorRed", got)
	}
}

func TestTTFBColor_Red(t *testing.T) {
	for _, d := range []time.Duration{501 * time.Millisecond, 2 * time.Second} {
		if got := ttfbColor(d); got != colorRed {
			t.Errorf("ttfbColor(%v) = %q, want colorRed", d, got)
		}
	}
}

// ---------------------------------------------------------------------------
// statusColor
// ---------------------------------------------------------------------------

func TestStatusColor_2xx(t *testing.T) {
	for _, code := range []int{200, 201, 204, 299} {
		if got := statusColor(code); got != colorGreen {
			t.Errorf("statusColor(%d) = %q, want colorGreen", code, got)
		}
	}
}

func TestStatusColor_3xx(t *testing.T) {
	for _, code := range []int{301, 302, 304, 399} {
		if got := statusColor(code); got != colorYellow {
			t.Errorf("statusColor(%d) = %q, want colorYellow", code, got)
		}
	}
}

func TestStatusColor_4xx(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 499} {
		if got := statusColor(code); got != colorRed {
			t.Errorf("statusColor(%d) = %q, want colorRed", code, got)
		}
	}
}

func TestStatusColor_5xx(t *testing.T) {
	for _, code := range []int{500, 502, 503, 599} {
		if got := statusColor(code); got != colorRed {
			t.Errorf("statusColor(%d) = %q, want colorRed", code, got)
		}
	}
}

// ---------------------------------------------------------------------------
// formatDuration
// ---------------------------------------------------------------------------

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0.00ms"},
		{time.Millisecond, "1.00ms"},
		{500 * time.Microsecond, "0.50ms"},
		{1500 * time.Microsecond, "1.50ms"},
		{time.Second, fmt.Sprintf("%.2fms", 1000.0)},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// printWaterfall — golden-output substring checks
// ---------------------------------------------------------------------------

// captureStdout redirects os.Stdout for the duration of fn and returns what was written.
func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = orig
	b, _ := io.ReadAll(r)
	return string(b)
}

func TestPrintWaterfall_BasicOutput(t *testing.T) {
	now := time.Now()
	result := ProbeResult{
		URL:          "http://example.com/",
		HTTPProtocol: "HTTP/1.1",
		StatusCode:   200,
		Status:       "200 OK",
		Timing: Timing{
			DNSLookup:        10 * time.Millisecond,
			TCPConnection:    20 * time.Millisecond,
			ServerProcessing: 150 * time.Millisecond,
			ContentTransfer:  70 * time.Millisecond,
			Total:            250 * time.Millisecond,
		},
		StartTime: now,
	}

	out := captureStdout(func() { printWaterfall(result, false) })

	for _, want := range []string{"http://example.com/", "200 OK", "DNS Lookup", "TCP Connection", "TTFB", "TTLB", "Total"} {
		if !strings.Contains(out, want) {
			t.Errorf("printWaterfall output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestPrintWaterfall_WithTrace(t *testing.T) {
	now := time.Now()
	result := ProbeResult{
		URL:          "http://example.com/trace",
		HTTPProtocol: "HTTP/2.0",
		StatusCode:   200,
		Status:       "200 OK",
		Timing: Timing{
			DNSLookup:        5 * time.Millisecond,
			TCPConnection:    10 * time.Millisecond,
			ServerProcessing: 100 * time.Millisecond,
			ContentTransfer:  40 * time.Millisecond,
			Total:            155 * time.Millisecond,
		},
		StartTime: now,
		TraceMessages: []TraceEvent{
			{Time: now, Text: "DNS lookup starting for example.com"},
			{Time: now.Add(5 * time.Millisecond), Text: "Connection attempt to 93.184.216.34:80"},
		},
	}

	out := captureStdout(func() { printWaterfall(result, true) })

	for _, want := range []string{"Trace", "DNS lookup starting for example.com", "Connection attempt"} {
		if !strings.Contains(out, want) {
			t.Errorf("printWaterfall(showTrace=true) missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestPrintWaterfall_WithRedirect(t *testing.T) {
	now := time.Now()
	result := ProbeResult{
		URL:          "http://example.com/final",
		HTTPProtocol: "HTTP/1.1",
		StatusCode:   200,
		Status:       "200 OK",
		Timing: Timing{
			DNSLookup:        5 * time.Millisecond,
			TCPConnection:    10 * time.Millisecond,
			ServerProcessing: 80 * time.Millisecond,
			ContentTransfer:  30 * time.Millisecond,
			Total:            125 * time.Millisecond,
		},
		StartTime: now.Add(50 * time.Millisecond),
		Redirects: []RedirectInfo{
			{
				URL:        "http://example.com/",
				StatusCode: 301,
				Status:     "301 Moved Permanently",
				Protocol:   "HTTP/1.1",
				StartTime:  now,
				EndTime:    now.Add(50 * time.Millisecond),
				Timing: Timing{
					DNSLookup:        5 * time.Millisecond,
					TCPConnection:    10 * time.Millisecond,
					ServerProcessing: 30 * time.Millisecond,
					Total:            50 * time.Millisecond,
				},
			},
		},
	}

	out := captureStdout(func() { printWaterfall(result, false) })

	for _, want := range []string{"301 Moved Permanently", "http://example.com/", "200 OK", "http://example.com/final", "DNS Lookup", "TCP Connection", "TTFB", "Total"} {
		if !strings.Contains(out, want) {
			t.Errorf("printWaterfall(with redirect) missing %q\nfull output:\n%s", want, out)
		}
	}
}

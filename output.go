package main

import (
	"fmt"
	"strings"
	"time"
)

const barWidth = 40

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

func renderBar(d, grandTotal time.Duration, color string) string {
	filled := 0
	if grandTotal > 0 && d > 0 {
		filled = int(float64(d) / float64(grandTotal) * barWidth)
		if filled > barWidth {
			filled = barWidth
		}
	}
	return color + strings.Repeat("█", filled) + colorDim + strings.Repeat("░", barWidth-filled) + colorReset
}

func ttfbColor(d time.Duration) string {
	switch {
	case d < 200*time.Millisecond:
		return colorGreen
	case d < 500*time.Millisecond:
		return colorYellow
	default:
		return colorRed
	}
}

func statusColor(code int) string {
	switch {
	case code >= 500:
		return colorRed
	case code >= 400:
		return colorRed
	case code >= 300:
		return colorYellow
	default:
		return colorGreen
	}
}

func printPhase(label string, d, grandTotal time.Duration, color string) {
	fmt.Printf("  %-18s %s  %s\n", label, renderBar(d, grandTotal, color), formatDuration(d))
}

func printHopHeader(marker, url string, statusCode int, status, protocol, connection string) {
	sc := statusColor(statusCode)
	meta := colorDim + protocol
	if connection != "" {
		meta += "  " + connection + " connection"
	}
	meta += colorReset
	fmt.Printf("\n%s%s%s  %s%s%s  %s  %s\n",
		colorBold, marker, colorReset,
		colorBold, url, colorReset,
		sc+status+colorReset,
		meta,
	)
}

func printHopPhases(t Timing, grandTotal time.Duration, showTTLB bool) {
	if !t.ReusedConnection {
		printPhase("DNS Lookup", t.DNSLookup, grandTotal, colorCyan)
		printPhase("TCP Connection", t.TCPConnection, grandTotal, colorCyan)
		printPhase("TLS Handshake", t.TLSHandshake, grandTotal, colorYellow)
	}
	printPhase("TTFB", t.ServerProcessing, grandTotal, ttfbColor(t.ServerProcessing))
	if showTTLB {
		printPhase("TTLB", t.ContentTransfer, grandTotal, colorBlue)
	}
}

func eventPhaseColor(text string, ttfb time.Duration) string {
	switch {
	case strings.Contains(text, "DNS") || strings.Contains(text, "Connection attempt"):
		return colorCyan
	case strings.Contains(text, "TLS"):
		return colorYellow
	case strings.Contains(text, "TTFB") || strings.Contains(text, "First response byte"):
		return ttfbColor(ttfb)
	case strings.Contains(text, "TTLB") || strings.Contains(text, "Response body"):
		return colorBlue
	default:
		return colorDim
	}
}

func printHopTrace(events []TraceEvent, marker, url string, statusCode int, status string, ttfb time.Duration, probeStart time.Time, lastTime *time.Time) {
	sc := statusColor(statusCode)
	fmt.Printf("\n  %s%s%s  %s%s%s  %s%s%s\n",
		colorBold, marker, colorReset,
		colorBold, url, colorReset,
		sc, status, colorReset,
	)
	for _, e := range events {
		elapsed := e.Time.Sub(probeStart)
		delta := e.Time.Sub(*lastTime)
		color := eventPhaseColor(e.Text, ttfb)
		fmt.Printf("  %s%7s%s  %s%-10s%s  %s%s%s\n",
			colorDim, fmt.Sprintf("+%dms", elapsed.Milliseconds()), colorReset,
			colorDim, fmt.Sprintf("[+%dms]", delta.Milliseconds()), colorReset,
			color, e.Text, colorReset,
		)
		*lastTime = e.Time
	}
}

func printWaterfall(result ProbeResult, showTrace bool) {
	grandTotal := result.Timing.Total
	if len(result.Redirects) > 0 {
		grandTotal += result.StartTime.Sub(result.Redirects[0].StartTime)
	}

	for _, r := range result.Redirects {
		connLabel := "new"
		if r.Timing.ReusedConnection {
			connLabel = "reused"
		}
		printHopHeader("→", r.URL, r.StatusCode, r.Status, r.Protocol, connLabel)
		printHopPhases(r.Timing, grandTotal, false)
	}

	connLabel := "new"
	if result.Timing.ReusedConnection {
		connLabel = "reused"
	}
	printHopHeader("●", result.URL, result.StatusCode, result.Status, result.HTTPProtocol, connLabel)
	printHopPhases(result.Timing, grandTotal, true)

	fmt.Printf("\n  %-18s %s  %s\n",
		colorBold+"Total"+colorReset,
		colorDim+strings.Repeat("█", barWidth)+colorReset,
		colorBold+formatDuration(grandTotal)+colorReset,
	)

	if showTrace {
		// Find probe start: the time of the very first event across all hops
		var probeStart time.Time
		if len(result.Redirects) > 0 && len(result.Redirects[0].TraceMessages) > 0 {
			probeStart = result.Redirects[0].TraceMessages[0].Time
		} else if len(result.TraceMessages) > 0 {
			probeStart = result.TraceMessages[0].Time
		}

		if !probeStart.IsZero() {
			fmt.Printf("\n%sTrace%s\n", colorBold, colorReset)
			lastTime := probeStart
			for _, r := range result.Redirects {
				printHopTrace(r.TraceMessages, "→", r.URL, r.StatusCode, r.Status, r.Timing.ServerProcessing, probeStart, &lastTime)
			}
			printHopTrace(result.TraceMessages, "●", result.URL, result.StatusCode, result.Status, result.Timing.ServerProcessing, probeStart, &lastTime)
		}
	}
}

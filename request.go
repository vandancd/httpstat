package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"
)

// Global variable to store all trace messages in chronological order
var globalTraceMessages []string

// handleRedirect handles HTTP redirects and collects timing information
func handleRedirect(req *http.Request, via []*http.Request, redirects *[]RedirectInfo, maxRedirects int) error {
	lastResponse := req.Response
	if lastResponse != nil {
		var currentTiming *Timing
		if timing, ok := lastResponse.Request.Context().Value(timingContextKey{}).(*Timing); ok {
			currentTiming = timing

			// Append current trace messages to global list
			globalTraceMessages = append(globalTraceMessages, traceMessages...)

			redirectInfo := RedirectInfo{
				URL:        lastResponse.Request.URL.String(),
				StatusCode: lastResponse.StatusCode,
				Status:     lastResponse.Status,
				StartTime:  lastResponse.Request.Context().Value(startTimeContextKey{}).(time.Time),
				EndTime:    time.Now(),
				Timing:     *currentTiming,
			}
			*redirects = append(*redirects, redirectInfo)

			// Reset trace messages and deduplication state for next request
			traceMessages = make([]string, 0)
			lastMessage = ""
			lastMessageTime = time.Time{}

			// Create a new timing object for the next request
			nextTiming := &Timing{}
			trace := createTracer(nextTiming)
			newCtx := context.WithValue(
				context.WithValue(
					httptrace.WithClientTrace(req.Context(), trace),
					startTimeContextKey{},
					time.Now(),
				),
				timingContextKey{},
				nextTiming,
			)
			*req = *req.WithContext(newCtx)
		}
	}

	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects (max: %d)", len(via), maxRedirects)
	}
	return nil
}

// createRequest creates a new HTTP request with tracing enabled
func createRequest(url string, timing *Timing) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	trace := createTracer(timing)
	req = req.WithContext(
		context.WithValue(
			context.WithValue(
				httptrace.WithClientTrace(req.Context(), trace),
				startTimeContextKey{},
				time.Now(),
			),
			timingContextKey{},
			timing,
		),
	)

	return req, nil
}

// processResponseBody reads the response body and updates timing information
func processResponseBody(resp *http.Response, timing *Timing, bodyStart, start time.Time) error {
	_, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return err
	}
	timing.ContentTransfer = time.Since(bodyStart)
	timing.Total = time.Since(start)
	return nil
}

// formatDuration formats a duration in milliseconds with 2 decimal places
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
}

// TimingJSON represents timing information in JSON format
type TimingJSON struct {
	DNSLookup     string `json:"dns_lookup,omitempty"`
	TCPConnection string `json:"tcp_connection,omitempty"`
	TLSHandshake  string `json:"tls_handshake,omitempty"`
	TTFB          string `json:"ttfb"`
	TTLB          string `json:"ttlb"`
	TotalTime     string `json:"total_time"`
}

// RedirectJSON represents a single redirect in JSON format
type RedirectJSON struct {
	URL        string     `json:"url"`
	StatusCode int        `json:"status_code"`
	Status     string     `json:"status"`
	Connection string     `json:"connection"`
	Timing     TimingJSON `json:"timing"`
}

// RedirectsJSON represents redirect information in JSON format
type RedirectsJSON struct {
	Count     int            `json:"count"`
	TotalTime string         `json:"total_time"`
	Chain     []RedirectJSON `json:"chain"`
}

// TotalTimesJSON represents total timing information in JSON format
type TotalTimesJSON struct {
	DNSLookups        string `json:"dns_lookups"`
	TCPConnections    string `json:"tcp_connections"`
	TLSHandshakes     string `json:"tls_handshakes"`
	TotalResponseTime string `json:"total_response_time"`
}

// TraceJSON represents trace information in JSON format
type TraceJSON struct {
	Messages []string `json:"messages"`
}

// ResponseJSON represents the complete HTTP response information in JSON format
type ResponseJSON struct {
	URL          string         `json:"url"`
	HTTPProtocol string         `json:"http_protocol"`
	StatusCode   int            `json:"status_code"`
	Status       string         `json:"status"`
	Connection   string         `json:"connection"`
	Timing       TimingJSON     `json:"timing"`
	Redirects    RedirectsJSON  `json:"redirects,omitempty"`
	Totals       TotalTimesJSON `json:"totals"`
	Trace        TraceJSON      `json:"trace"`
}

// printResults prints the final results of the HTTP request in JSON format
func printResults(resp *http.Response, redirects []RedirectInfo, finalTiming Timing) {
	// Append final trace messages to global list
	globalTraceMessages = append(globalTraceMessages, traceMessages...)

	result := ResponseJSON{
		URL:          resp.Request.URL.String(),
		HTTPProtocol: resp.Proto,
		StatusCode:   resp.StatusCode,
		Status:       resp.Status,
		Connection:   connectionInfo(finalTiming.ReusedConnection),
		Timing: TimingJSON{
			TTFB:      formatDuration(finalTiming.ServerProcessing),
			TTLB:      formatDuration(finalTiming.ContentTransfer),
			TotalTime: formatDuration(finalTiming.Total),
		},
		Trace: TraceJSON{
			Messages: globalTraceMessages,
		},
	}

	if !finalTiming.ReusedConnection {
		result.Timing.DNSLookup = formatDuration(finalTiming.DNSLookup)
		result.Timing.TCPConnection = formatDuration(finalTiming.TCPConnection)
		result.Timing.TLSHandshake = formatDuration(finalTiming.TLSHandshake)
	}

	// Calculate redirect information
	if len(redirects) > 0 {
		var totalRedirectTime time.Duration
		redirectChain := make([]RedirectJSON, 0, len(redirects))

		for _, redirect := range redirects {
			totalRedirectTime += redirect.EndTime.Sub(redirect.StartTime)
			redirectJSON := RedirectJSON{
				URL:        redirect.URL,
				StatusCode: redirect.StatusCode,
				Status:     redirect.Status,
				Connection: connectionInfo(redirect.Timing.ReusedConnection),
				Timing: TimingJSON{
					TTFB:      formatDuration(redirect.Timing.ServerProcessing),
					TotalTime: formatDuration(redirect.EndTime.Sub(redirect.StartTime)),
				},
			}

			if !redirect.Timing.ReusedConnection {
				redirectJSON.Timing.DNSLookup = formatDuration(redirect.Timing.DNSLookup)
				redirectJSON.Timing.TCPConnection = formatDuration(redirect.Timing.TCPConnection)
				redirectJSON.Timing.TLSHandshake = formatDuration(redirect.Timing.TLSHandshake)
			}

			redirectChain = append(redirectChain, redirectJSON)
		}

		result.Redirects = RedirectsJSON{
			Count:     len(redirects),
			TotalTime: formatDuration(totalRedirectTime),
			Chain:     redirectChain,
		}
	}

	// Calculate total times
	var totalDNS, totalTCP, totalTLS time.Duration
	for _, redirect := range redirects {
		if !redirect.Timing.ReusedConnection {
			totalDNS += redirect.Timing.DNSLookup
			totalTCP += redirect.Timing.TCPConnection
			totalTLS += redirect.Timing.TLSHandshake
		}
	}
	if !finalTiming.ReusedConnection {
		totalDNS += finalTiming.DNSLookup
		totalTCP += finalTiming.TCPConnection
		totalTLS += finalTiming.TLSHandshake
	}

	// Calculate total response time
	var totalResponseTime time.Duration
	if len(redirects) > 0 {
		firstRedirect := redirects[0]
		totalResponseTime = finalTiming.Total + resp.Request.Context().Value(startTimeContextKey{}).(time.Time).Sub(firstRedirect.StartTime)
	} else {
		totalResponseTime = finalTiming.Total
	}

	result.Totals = TotalTimesJSON{
		DNSLookups:        formatDuration(totalDNS),
		TCPConnections:    formatDuration(totalTCP),
		TLSHandshakes:     formatDuration(totalTLS),
		TotalResponseTime: formatDuration(totalResponseTime),
	}

	// Output JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

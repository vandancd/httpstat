package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	probe "github.com/vandancd/httpstat/pkg/probe"
)

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
}

type TimingJSON struct {
	DNSLookup     string `json:"dns_lookup,omitempty"`
	TCPConnection string `json:"tcp_connection,omitempty"`
	TLSHandshake  string `json:"tls_handshake,omitempty"`
	TTFB          string `json:"ttfb"`
	TTLB          string `json:"ttlb"`
	TotalTime     string `json:"total_time"`
}

type RedirectJSON struct {
	URL        string     `json:"url"`
	StatusCode int        `json:"status_code"`
	Status     string     `json:"status"`
	Connection string     `json:"connection"`
	Timing     TimingJSON `json:"timing"`
}

type RedirectsJSON struct {
	Count     int            `json:"count"`
	TotalTime string         `json:"total_time"`
	Chain     []RedirectJSON `json:"chain"`
}

type TotalTimesJSON struct {
	DNSLookups        string `json:"dns_lookups"`
	TCPConnections    string `json:"tcp_connections"`
	TLSHandshakes     string `json:"tls_handshakes"`
	TotalResponseTime string `json:"total_response_time"`
}

type TraceJSON struct {
	Messages []string `json:"messages"`
}

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

func formatTraceEvents(p probe.Result) []string {
	var out []string
	for _, r := range p.Redirects {
		for _, e := range r.TraceMessages {
			out = append(out, e.Time.Format("2006-01-02 15:04:05.000")+": "+e.Text)
		}
	}
	for _, e := range p.TraceMessages {
		out = append(out, e.Time.Format("2006-01-02 15:04:05.000")+": "+e.Text)
	}
	return out
}

func printJSON(p probe.Result) {
	connLabel := "new"
	if p.Timing.ReusedConnection {
		connLabel = "reused"
	}
	result := ResponseJSON{
		URL:          p.URL,
		HTTPProtocol: p.HTTPProtocol,
		StatusCode:   p.StatusCode,
		Status:       p.Status,
		Connection:   connLabel,
		Timing: TimingJSON{
			TTFB:      formatDuration(p.Timing.ServerProcessing),
			TTLB:      formatDuration(p.Timing.ContentTransfer),
			TotalTime: formatDuration(p.Timing.Total),
		},
		Trace: TraceJSON{
			Messages: formatTraceEvents(p),
		},
	}

	if !p.Timing.ReusedConnection {
		result.Timing.DNSLookup = formatDuration(p.Timing.DNSLookup)
		result.Timing.TCPConnection = formatDuration(p.Timing.TCPConnection)
		result.Timing.TLSHandshake = formatDuration(p.Timing.TLSHandshake)
	}

	if len(p.Redirects) > 0 {
		var totalRedirectTime time.Duration
		redirectChain := make([]RedirectJSON, 0, len(p.Redirects))

		for _, redirect := range p.Redirects {
			totalRedirectTime += redirect.EndTime.Sub(redirect.StartTime)
			redirectConnLabel := "new"
			if redirect.Timing.ReusedConnection {
				redirectConnLabel = "reused"
			}
			redirectJSON := RedirectJSON{
				URL:        redirect.URL,
				StatusCode: redirect.StatusCode,
				Status:     redirect.Status,
				Connection: redirectConnLabel,
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
			Count:     len(p.Redirects),
			TotalTime: formatDuration(totalRedirectTime),
			Chain:     redirectChain,
		}
	}

	var totalDNS, totalTCP, totalTLS time.Duration
	for _, redirect := range p.Redirects {
		if !redirect.Timing.ReusedConnection {
			totalDNS += redirect.Timing.DNSLookup
			totalTCP += redirect.Timing.TCPConnection
			totalTLS += redirect.Timing.TLSHandshake
		}
	}
	if !p.Timing.ReusedConnection {
		totalDNS += p.Timing.DNSLookup
		totalTCP += p.Timing.TCPConnection
		totalTLS += p.Timing.TLSHandshake
	}

	var totalResponseTime time.Duration
	if len(p.Redirects) > 0 {
		totalResponseTime = p.Timing.Total + p.StartTime.Sub(p.Redirects[0].StartTime)
	} else {
		totalResponseTime = p.Timing.Total
	}

	result.Totals = TotalTimesJSON{
		DNSLookups:        formatDuration(totalDNS),
		TCPConnections:    formatDuration(totalTCP),
		TLSHandshakes:     formatDuration(totalTLS),
		TotalResponseTime: formatDuration(totalResponseTime),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

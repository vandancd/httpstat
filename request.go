package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type RedirectHandler struct {
	log          *TraceLog
	maxRedirects int
	useCustomDNS bool
	redirects    *[]RedirectInfo
}

func (h *RedirectHandler) Handle(req *http.Request, via []*http.Request) error {
	if lastResponse := req.Response; lastResponse != nil {
		if m := measurementFromContext(lastResponse.Request.Context()); m != nil {
			*h.redirects = append(*h.redirects, RedirectInfo{
				URL:           lastResponse.Request.URL.String(),
				StatusCode:    lastResponse.StatusCode,
				Status:        lastResponse.Status,
				Protocol:      lastResponse.Proto,
				StartTime:     m.StartTime(),
				EndTime:       time.Now(),
				Timing:        m.Result(),
				TraceMessages: h.log.Flush(),
			})
			nextM := NewMeasurement(h.log, h.useCustomDNS)
			*req = *req.WithContext(nextM.Instrument(req.Context()))
		}
	}

	if len(via) >= h.maxRedirects {
		return fmt.Errorf("stopped after %d redirects (max: %d)", len(via), h.maxRedirects)
	}
	return nil
}

func createRequest(url string, m *Measurement, userAgent string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	return req.WithContext(m.Instrument(req.Context())), nil
}

func processResponseBody(resp *http.Response, m *Measurement, bodyStart time.Time) error {
	_, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return err
	}
	m.FinishBody(bodyStart)
	return nil
}

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

func formatTraceEvents(probe ProbeResult) []string {
	var out []string
	for _, r := range probe.Redirects {
		for _, e := range r.TraceMessages {
			out = append(out, e.Time.Format("2006-01-02 15:04:05.000")+": "+e.Text)
		}
	}
	for _, e := range probe.TraceMessages {
		out = append(out, e.Time.Format("2006-01-02 15:04:05.000")+": "+e.Text)
	}
	return out
}

func printJSON(probe ProbeResult) {
	connLabel := "new"
	if probe.Timing.ReusedConnection {
		connLabel = "reused"
	}
	result := ResponseJSON{
		URL:          probe.URL,
		HTTPProtocol: probe.HTTPProtocol,
		StatusCode:   probe.StatusCode,
		Status:       probe.Status,
		Connection:   connLabel,
		Timing: TimingJSON{
			TTFB:      formatDuration(probe.Timing.ServerProcessing),
			TTLB:      formatDuration(probe.Timing.ContentTransfer),
			TotalTime: formatDuration(probe.Timing.Total),
		},
		Trace: TraceJSON{
			Messages: formatTraceEvents(probe),
		},
	}

	if !probe.Timing.ReusedConnection {
		result.Timing.DNSLookup = formatDuration(probe.Timing.DNSLookup)
		result.Timing.TCPConnection = formatDuration(probe.Timing.TCPConnection)
		result.Timing.TLSHandshake = formatDuration(probe.Timing.TLSHandshake)
	}

	if len(probe.Redirects) > 0 {
		var totalRedirectTime time.Duration
		redirectChain := make([]RedirectJSON, 0, len(probe.Redirects))

		for _, redirect := range probe.Redirects {
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
			Count:     len(probe.Redirects),
			TotalTime: formatDuration(totalRedirectTime),
			Chain:     redirectChain,
		}
	}

	var totalDNS, totalTCP, totalTLS time.Duration
	for _, redirect := range probe.Redirects {
		if !redirect.Timing.ReusedConnection {
			totalDNS += redirect.Timing.DNSLookup
			totalTCP += redirect.Timing.TCPConnection
			totalTLS += redirect.Timing.TLSHandshake
		}
	}
	if !probe.Timing.ReusedConnection {
		totalDNS += probe.Timing.DNSLookup
		totalTCP += probe.Timing.TCPConnection
		totalTLS += probe.Timing.TLSHandshake
	}

	var totalResponseTime time.Duration
	if len(probe.Redirects) > 0 {
		totalResponseTime = probe.Timing.Total + probe.StartTime.Sub(probe.Redirects[0].StartTime)
	} else {
		totalResponseTime = probe.Timing.Total
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

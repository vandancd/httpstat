package probe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func createRequest(ctx context.Context, url string, m *Measurement, opts Options) (*http.Request, error) {
	method := opts.Method
	if method == "" {
		method = "GET"
	}

	var body io.Reader
	if opts.JSONBody != "" {
		body = strings.NewReader(opts.JSONBody)
	} else if opts.Body != "" {
		body = strings.NewReader(opts.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	if opts.JSONBody != "" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
	}
	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}
	if opts.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+opts.BearerToken)
	}
	for _, h := range opts.Headers {
		parts := strings.SplitN(h, ":", 2)
		req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
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

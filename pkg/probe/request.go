package probe

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

func createRequest(ctx context.Context, url string, m *Measurement, userAgent string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func newBrowserSession(parent context.Context) (ctx context.Context, cleanup func(), err error) {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
	}
	switch runtime.GOOS {
	case "windows":
		opts = append(opts, chromedp.ExecPath(`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`))
	case "darwin":
		opts = append(opts, chromedp.ExecPath(`/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge`))
	case "linux":
		opts = append(opts, chromedp.ExecPath(`/usr/bin/microsoft-edge`))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(parent, opts...)
	browseCtx, browseCancel := chromedp.NewContext(allocCtx)

	if err := chromedp.Run(browseCtx, network.Enable()); err != nil {
		browseCancel()
		allocCancel()
		return nil, nil, err
	}

	return browseCtx, func() {
		browseCancel()
		allocCancel()
	}, nil
}

// extractBrowserTiming computes formatted timing strings from a raw performance.timing map.
// Pure function — no browser dependency; testable with fixture data.
func extractBrowserTiming(raw map[string]interface{}) map[string]string {
	get := func(key string) float64 {
		if v, ok := raw[key]; ok {
			if f, ok := v.(float64); ok {
				return f
			}
		}
		return 0
	}

	dns := get("domainLookupEnd") - get("domainLookupStart")
	tcp := get("connectEnd") - get("connectStart")
	tls := 0.0
	if val, ok := raw["secureConnectionStart"].(float64); ok && val > 0 {
		tls = get("connectEnd") - val
	}
	ttfb := get("responseStart") - get("requestStart")
	ttlb := get("responseEnd") - get("requestStart")
	domInteractive := get("domInteractive") - get("navigationStart")
	domComplete := get("domComplete") - get("navigationStart")
	total := get("loadEventEnd") - get("navigationStart")

	return map[string]string{
		"dns_lookup":          fmt.Sprintf("%.2fms", dns),
		"tcp_connection":      fmt.Sprintf("%.2fms", tcp),
		"tls_handshake":       fmt.Sprintf("%.2fms", tls),
		"ttfb":                fmt.Sprintf("%.2fms", ttfb),
		"ttlb":                fmt.Sprintf("%.2fms", ttlb),
		"dom_interactive":     fmt.Sprintf("%.2fms", domInteractive),
		"dom_complete":        fmt.Sprintf("%.2fms", domComplete),
		"total_response_time": fmt.Sprintf("%.2fms", total),
	}
}

func formatBrowserOutput(timing map[string]string, response map[string]interface{}, networkErrors []map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(map[string]interface{}{
		"timing":         timing,
		"network_errors": networkErrors,
		"response":       response,
	}, "", "  ")
}

func runBrowserProbe(url string) error {
	ctx, cleanup, err := newBrowserSession(context.Background())
	if err != nil {
		return err
	}
	defer cleanup()

	var mainResponse map[string]interface{}
	networkErrors := []map[string]interface{}{}

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventResponseReceived:
			if e.Type == network.ResourceTypeDocument {
				mainResponse = map[string]interface{}{
					"status":     e.Response.Status,
					"statusText": e.Response.StatusText,
					"protocol":   e.Response.Protocol,
					"url":        e.Response.URL,
				}
			}
		case *network.EventLoadingFailed:
			networkErrors = append(networkErrors, map[string]interface{}{
				"requestId":     e.RequestID.String(),
				"errorText":     e.ErrorText,
				"timestamp":     e.Timestamp,
				"type":          e.Type.String(),
				"canceled":      e.Canceled,
				"blockedReason": e.BlockedReason.String(),
			})
		}
	})

	var raw map[string]interface{}
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Evaluate(`new Promise(resolve => {
      window.addEventListener("load", () => resolve(true));
  })`, nil),
		chromedp.Evaluate(`performance.timing.toJSON()`, &raw),
	); err != nil {
		return err
	}

	outJSON, err := formatBrowserOutput(extractBrowserTiming(raw), mainResponse, networkErrors)
	if err != nil {
		return err
	}
	fmt.Println(string(outJSON))
	return nil
}

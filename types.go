package main

import "time"

// Timing holds timing information for various stages of the HTTP request
type Timing struct {
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	ContentTransfer  time.Duration
	Total            time.Duration
	ReusedConnection bool
}

// RedirectInfo holds information about a redirect
type RedirectInfo struct {
	URL           string
	StatusCode    int
	Status        string
	StartTime     time.Time
	EndTime       time.Time
	Timing        Timing
	TraceMessages []string
}

// Context keys for storing values in request context
type startTimeContextKey struct{}
type timingContextKey struct{}

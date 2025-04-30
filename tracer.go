package main

import (
	"crypto/tls"
	"fmt"
	"net/http/httptrace"
	"strings"
	"time"
)

// traceMessages stores trace messages during the request
var traceMessages []string
var lastMessage string
var lastMessageTime time.Time

// addTraceMessage adds a message to the trace log
func addTraceMessage(format string, args ...interface{}) {
	now := time.Now()
	// If you want to print timestamp, uncomment the following next two lines.
	timestamp := now.Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf("%s: %s", timestamp, fmt.Sprintf(format, args...))

	//msg := fmt.Sprintf(format, args...)

	// Deduplicate messages that occur within 10ms of each other
	//now := time.Now()
	if msg == lastMessage && now.Sub(lastMessageTime) < 10*time.Millisecond {
		return
	}

	traceMessages = append(traceMessages, msg)
	lastMessage = msg
	lastMessageTime = now
}

// createTracer creates a new trace with timing information
func createTracer(timing *Timing) *httptrace.ClientTrace {
	var start, connect, dns, tlsHandshake time.Time
	var firstByte time.Time

	return &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) {
			dns = time.Now()
			// Get system DNS servers if not using custom ones
			if resolver == nil {
				if servers := getSystemDNSServers(); len(servers) > 0 {
					addTraceMessage("Using system DNS servers: %s", strings.Join(servers, ", "))
				}
			}
			addTraceMessage("DNS lookup starting for %s", dsi.Host)
		},
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			timing.DNSLookup = time.Since(dns)
		},
		ConnectStart: func(network, addr string) {
			connect = time.Now()
			addTraceMessage("Connection attempt to %s", addr)
		},
		ConnectDone: func(network, addr string, err error) {
			timing.TCPConnection = time.Since(connect)
		},
		TLSHandshakeStart: func() {
			tlsHandshake = time.Now()
			addTraceMessage("TLS handshake starting")
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			timing.TLSHandshake = time.Since(tlsHandshake)
			if err != nil {
				addTraceMessage("TLS handshake failed: %v", err)
			} else {
				addTraceMessage("TLS handshake completed")
			}
		},
		GotFirstResponseByte: func() {
			firstByte = time.Now()
			timing.ServerProcessing = firstByte.Sub(start)
			addTraceMessage("First response byte received (TTFB)")
		},
		GetConn: func(hostPort string) {
			start = time.Now()
			addTraceMessage("Getting connection for %s", hostPort)
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			addTraceMessage("Got connection: reused=%v, was_idle=%v, idle_time=%v",
				connInfo.Reused, connInfo.WasIdle, connInfo.IdleTime)
			timing.ReusedConnection = connInfo.Reused
			if connInfo.Reused {
				// Reset timing information for reused connections
				timing.DNSLookup = 0
				timing.TCPConnection = 0
				timing.TLSHandshake = 0
			}
		},
	}
}

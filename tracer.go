package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http/httptrace"
	"strings"
	"time"
)

type measurementContextKey struct{}

type Measurement struct {
	timing       Timing
	start        time.Time
	log          *TraceLog
	useCustomDNS bool
}

func NewMeasurement(log *TraceLog, useCustomDNS bool) *Measurement {
	return &Measurement{log: log, start: time.Now(), useCustomDNS: useCustomDNS}
}

// Instrument wires httptrace callbacks into ctx and stores m in context.
func (m *Measurement) Instrument(ctx context.Context) context.Context {
	return context.WithValue(httptrace.WithClientTrace(ctx, createTracer(m)), measurementContextKey{}, m)
}

// FinishBody records ContentTransfer (from bodyStart to now) and Total (from m.start to now).
func (m *Measurement) FinishBody(bodyStart time.Time) {
	m.timing.ContentTransfer = time.Since(bodyStart)
	m.log.Log("Response body fully read (TTLB)")
	m.timing.Total = time.Since(m.start)
}

// Result returns the completed Timing.
func (m *Measurement) Result() Timing { return m.timing }

// StartTime returns when this measurement began.
func (m *Measurement) StartTime() time.Time { return m.start }

func measurementFromContext(ctx context.Context) *Measurement {
	m, _ := ctx.Value(measurementContextKey{}).(*Measurement)
	return m
}

type TraceLog struct {
	messages        []string
	lastMessage     string
	lastMessageTime time.Time
}

func (l *TraceLog) Log(format string, args ...interface{}) {
	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf("%s: %s", timestamp, fmt.Sprintf(format, args...))
	// Deduplicate messages that occur within 10ms of each other
	if msg == l.lastMessage && now.Sub(l.lastMessageTime) < 10*time.Millisecond {
		return
	}
	l.messages = append(l.messages, msg)
	l.lastMessage = msg
	l.lastMessageTime = now
}

func (l *TraceLog) Flush() []string {
	msgs := l.messages
	l.messages = nil
	l.lastMessage = ""
	l.lastMessageTime = time.Time{}
	return msgs
}

func createTracer(m *Measurement) *httptrace.ClientTrace {
	var start, connect, dns, tlsHandshake time.Time
	var firstByte time.Time
	t := &m.timing
	log := m.log

	return &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) {
			dns = time.Now()
			if !m.useCustomDNS {
				if servers := getSystemDNSServers(); len(servers) > 0 {
					log.Log("Using system DNS servers: %s", strings.Join(servers, ", "))
				}
			}
			log.Log("DNS lookup starting for %s", dsi.Host)
		},
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			t.DNSLookup = time.Since(dns)
		},
		ConnectStart: func(network, addr string) {
			connect = time.Now()
			log.Log("Connection attempt to %s", addr)
		},
		ConnectDone: func(network, addr string, err error) {
			t.TCPConnection = time.Since(connect)
		},
		TLSHandshakeStart: func() {
			tlsHandshake = time.Now()
			log.Log("TLS handshake starting")
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			t.TLSHandshake = time.Since(tlsHandshake)
			if err != nil {
				log.Log("TLS handshake failed: %v", err)
			} else {
				log.Log("TLS handshake completed")
			}
		},
		GotFirstResponseByte: func() {
			firstByte = time.Now()
			t.ServerProcessing = firstByte.Sub(start)
			log.Log("First response byte received (TTFB)")
		},
		GetConn: func(hostPort string) {
			start = time.Now()
			log.Log("Getting connection for %s", hostPort)
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			log.Log("Got connection: reused=%v, was_idle=%v, idle_time=%v",
				connInfo.Reused, connInfo.WasIdle, connInfo.IdleTime)
			t.ReusedConnection = connInfo.Reused
			if connInfo.Reused {
				t.DNSLookup = 0
				t.TCPConnection = 0
				t.TLSHandshake = 0
			}
		},
	}
}

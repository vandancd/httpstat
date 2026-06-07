package probe

import (
	"context"
	"net"
	"net/http"
	"time"
)

type Timing struct {
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	ContentTransfer  time.Duration
	Total            time.Duration
	ReusedConnection bool
}

type RedirectInfo struct {
	URL           string
	StatusCode    int
	Status        string
	Protocol      string
	StartTime     time.Time
	EndTime       time.Time
	Timing        Timing
	TraceMessages []TraceEvent
}

type Result struct {
	URL           string
	HTTPProtocol  string
	StatusCode    int
	Status        string
	Redirects     []RedirectInfo
	Timing        Timing
	StartTime     time.Time
	TraceMessages []TraceEvent
}

type Options struct {
	UseHTTP1     bool
	UseHTTP11    bool
	NoKeepAlive  bool
	Timeout      time.Duration
	MaxRedirects int
	DNSServers   []string
	PreferIPv6   bool
	UserAgent    string
	Method       string
	Headers      []string
	BearerToken  string
	Body         string
	JSONBody     string
	// Optional seams for testing. When nil/zero, Run builds defaults from the
	// other fields. When provided, Run uses them directly.
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	Transport   http.RoundTripper
}

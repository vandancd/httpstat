package probe

import "time"

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
}

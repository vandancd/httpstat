package probe

import (
	"crypto/tls"
	"testing"
)

func TestCreateTransport_HTTP2_Default(t *testing.T) {
	tr := createTransport(false, false, false, nil)
	if !tr.ForceAttemptHTTP2 {
		t.Error("expected ForceAttemptHTTP2=true for default (HTTP/2) transport")
	}
	if tr.TLSClientConfig == nil {
		t.Error("expected TLSClientConfig to be set for HTTP/2 transport")
	}
	if tr.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS MinVersion=1.2, got %d", tr.TLSClientConfig.MinVersion)
	}
}

func TestCreateTransport_HTTP1_DisablesHTTP2(t *testing.T) {
	tr := createTransport(true, false, false, nil)
	if tr.ForceAttemptHTTP2 {
		t.Error("expected ForceAttemptHTTP2=false for HTTP/1.0 transport")
	}
	if len(tr.TLSNextProto) != 0 {
		t.Errorf("expected empty TLSNextProto for HTTP/1.0, got %d entries", len(tr.TLSNextProto))
	}
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.MaxVersion != tls.VersionTLS12 {
		t.Error("expected TLS MaxVersion=1.2 for HTTP/1.0 transport")
	}
}

func TestCreateTransport_HTTP11_DisablesHTTP2(t *testing.T) {
	tr := createTransport(false, true, false, nil)
	if tr.ForceAttemptHTTP2 {
		t.Error("expected ForceAttemptHTTP2=false for HTTP/1.1 transport")
	}
	if len(tr.TLSNextProto) != 0 {
		t.Errorf("expected empty TLSNextProto for HTTP/1.1, got %d entries", len(tr.TLSNextProto))
	}
}

func TestCreateTransport_NoKeepAlive(t *testing.T) {
	tr := createTransport(false, false, true, nil)
	if !tr.DisableKeepAlives {
		t.Error("expected DisableKeepAlives=true when noKeepAlive=true")
	}
}

func TestCreateTransport_KeepAliveDefault(t *testing.T) {
	tr := createTransport(false, false, false, nil)
	if tr.DisableKeepAlives {
		t.Error("expected DisableKeepAlives=false by default")
	}
}

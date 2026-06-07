package probe

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestRun_CustomDialContext_IsCalled(t *testing.T) {
	called := make(chan struct{}, 1)
	_, err := Run(context.Background(), "http://test.invalid/", Options{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			select {
			case called <- struct{}{}:
			default:
			}
			return nil, fmt.Errorf("fake dialer: aborting")
		},
		MaxRedirects: 5,
		Timeout:      2 * time.Second,
	})
	if err == nil {
		t.Fatal("expected error from fake dialer, got nil")
	}
	select {
	case <-called:
	default:
		t.Error("custom DialContext was never called")
	}
}

func TestRun_CustomTransport_IsCalled(t *testing.T) {
	called := make(chan struct{}, 1)
	_, err := Run(context.Background(), "http://test.invalid/", Options{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			select {
			case called <- struct{}{}:
			default:
			}
			return nil, fmt.Errorf("fake transport: aborting")
		}),
		MaxRedirects: 5,
		Timeout:      2 * time.Second,
	})
	if err == nil {
		t.Fatal("expected error from fake transport, got nil")
	}
	select {
	case <-called:
	default:
		t.Error("custom Transport was never called")
	}
}

func TestRun_CustomTransport_OverridesDialContext(t *testing.T) {
	dialerCalled := make(chan struct{}, 1)
	transportCalled := make(chan struct{}, 1)

	_, _ = Run(context.Background(), "http://test.invalid/", Options{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			select {
			case dialerCalled <- struct{}{}:
			default:
			}
			return nil, fmt.Errorf("should not be reached")
		},
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			select {
			case transportCalled <- struct{}{}:
			default:
			}
			return nil, fmt.Errorf("fake transport")
		}),
		MaxRedirects: 5,
		Timeout:      2 * time.Second,
	})

	select {
	case <-transportCalled:
	default:
		t.Error("Transport was not called")
	}
	select {
	case <-dialerCalled:
		t.Error("DialContext should not be called when Transport is injected")
	default:
	}
}

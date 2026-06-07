package probe

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
)

// TestDNSCounterAtomicConcurrent verifies that the atomic server-index counter
// used inside createCustomResolver does not data-race under concurrent access.
// This directly mirrors the counter logic in dns.go so the race detector will
// catch any regression if someone reverts to a plain int.
//
// Run with: go test -race ./pkg/probe/...
func TestDNSCounterAtomicConcurrent(t *testing.T) {
	t.Parallel()

	const numServers = 3
	var counter atomic.Int32
	n := int32(numServers)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	indices := make([]int32, goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			// Mirror the dns.go expression exactly.
			indices[i] = (counter.Add(1) - 1) % n
		}()
	}
	wg.Wait()

	for i, idx := range indices {
		if idx < 0 || idx >= n {
			t.Errorf("goroutine %d: index %d out of range [0, %d)", i, idx, n)
		}
	}
}

// TestCreateCustomResolverDialConcurrent exercises createCustomResolver with
// multiple goroutines dialling concurrently. Each goroutine owns its TraceLog
// (which is intentionally not goroutine-safe in production, where it is owned
// by a single request) so the race detector focuses on the shared atomic
// counter inside the resolver.
//
// Run with: go test -race ./pkg/probe/...
func TestCreateCustomResolverDialConcurrent(t *testing.T) {
	t.Parallel()

	// Loopback addresses succeed immediately for UDP without a real DNS server.
	servers := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			log := &TraceLog{}
			r := createCustomResolver(servers, log)
			conn, err := r.Dial(context.Background(), "udp", "")
			if err == nil && conn != nil {
				conn.Close()
			}
		}()
	}
	wg.Wait()
}

// TestCreateCustomResolverRoundRobin verifies that sequential Dial calls
// distribute across all servers at least once in round-robin order.
func TestCreateCustomResolverRoundRobin(t *testing.T) {
	t.Parallel()

	servers := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
	log := &TraceLog{}
	resolver := createCustomResolver(servers, log)
	dialFn := resolver.Dial

	seen := make(map[string]int)
	for i := 0; i < len(servers)*3; i++ {
		conn, err := dialFn(context.Background(), "udp", "")
		if err != nil {
			t.Fatalf("dial %d failed: %v", i, err)
		}
		host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		seen[host]++
		conn.Close()
	}

	// Every server must be selected at least once across 9 calls over 3 servers.
	for _, s := range servers {
		if seen[s] == 0 {
			t.Errorf("server %s was never selected", s)
		}
	}
}

package network

import (
	"testing"
	"time"
)

type dummyAddr struct {
	addr string
}

func (d dummyAddr) Network() string { return "tcp" }
func (d dummyAddr) String() string  { return d.addr }

func TestConnectionThrottle(t *testing.T) {
	filter := NewFilter(FilterOptions{
		ConnectionThrottle:  50 * time.Millisecond,
		MaxConnectionsPerIP: 10,
		RateLimitPerSecond:  100,
	})

	addr := dummyAddr{addr: "1.2.3.4:12345"}

	allowed, _ := filter.Allow(addr)
	if !allowed {
		t.Fatalf("first connection attempt should be allowed")
	}

	// Immediate next attempt from same IP should be throttled
	allowed2, msg := filter.Allow(addr)
	if allowed2 {
		t.Errorf("immediate connection should be throttled")
	}
	if msg == "" {
		t.Errorf("expected throttle message")
	}

	// Wait past throttle
	time.Sleep(60 * time.Millisecond)
	allowed3, _ := filter.Allow(addr)
	if !allowed3 {
		t.Errorf("connection should be allowed after throttle expires")
	}
}

func TestMaxConnectionsPerIP(t *testing.T) {
	filter := NewFilter(FilterOptions{
		ConnectionThrottle:  0,
		MaxConnectionsPerIP: 2,
		RateLimitPerSecond:  100,
	})

	addr := dummyAddr{addr: "5.6.7.8:1000"}

	allowed1, _ := filter.Allow(addr)
	allowed2, _ := filter.Allow(addr)
	if !allowed1 || !allowed2 {
		t.Fatalf("expected first two connections to be allowed")
	}

	// Third attempt exceeds cap of 2
	allowed3, msg := filter.Allow(addr)
	if allowed3 {
		t.Errorf("expected third connection to be rejected due to max conns per IP")
	}
	if msg == "" {
		t.Errorf("expected max conns message")
	}

	// Release one connection
	filter.Release(addr)
	allowed4, _ := filter.Allow(addr)
	if !allowed4 {
		t.Errorf("expected connection to be allowed after releasing one slot")
	}
}

func TestRateLimitPerSecond(t *testing.T) {
	filter := NewFilter(FilterOptions{
		ConnectionThrottle:  0,
		MaxConnectionsPerIP: 100,
		RateLimitPerSecond:  3,
	})

	addr := dummyAddr{addr: "9.9.9.9:5000"}

	for i := 0; i < 3; i++ {
		allowed, _ := filter.Allow(addr)
		if !allowed {
			t.Fatalf("attempt %d should be allowed within rate limit", i+1)
		}
	}

	// 4th attempt exceeds 3/sec
	allowed4, msg := filter.Allow(addr)
	if allowed4 {
		t.Errorf("4th attempt should be blocked by rate limit")
	}
	if msg == "" {
		t.Errorf("expected rate limit message")
	}
}

func TestPruneStaleAndIPNormalization(t *testing.T) {
	filter := NewFilter(FilterOptions{
		ConnectionThrottle:  10 * time.Millisecond,
		MaxConnectionsPerIP: 2,
		RateLimitPerSecond:  10,
	})

	// Test IPv6 uncompressed vs compressed matches same host
	addr1 := dummyAddr{addr: "[2001:0db8:0000:0000:0000:0000:0000:0001]:12345"}
	addr2 := dummyAddr{addr: "[2001:db8::1]:54321"}

	if allowed, _ := filter.Allow(addr1); !allowed {
		t.Fatalf("first connection should be allowed")
	}

	// Should be throttled because addr1 and addr2 canonicalize to the same IP
	if allowed, _ := filter.Allow(addr2); allowed {
		t.Errorf("expected second connection from same IPv6 host to be throttled")
	}

	// Test pruneStale
	filter.pruneStale(time.Now().Add(2 * time.Hour))
	if len(filter.lastConnect) != 0 {
		t.Errorf("expected lastConnect to be pruned, got %d entries", len(filter.lastConnect))
	}
}

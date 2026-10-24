package network

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// FilterOptions configures the Paper-grade connection security filter.
type FilterOptions struct {
	// ConnectionThrottle is the minimum duration required between connection attempts
	// from the same IP address. Default is 1000ms. Set to 0 to disable.
	ConnectionThrottle time.Duration
	// MaxConnectionsPerIP is the maximum number of concurrent active connections
	// permitted from a single IP address. Default is 5. Set to 0 for unlimited.
	MaxConnectionsPerIP int
	// RateLimitPerSecond is the maximum connection attempts allowed within a 1-second window.
	RateLimitPerSecond int
}

// DefaultFilterOptions returns standard production-grade connection limits.
func DefaultFilterOptions() FilterOptions {
	return FilterOptions{
		ConnectionThrottle:  1000 * time.Millisecond,
		MaxConnectionsPerIP: 5,
		RateLimitPerSecond:  10,
	}
}

type rateTracker struct {
	windowStart time.Time
	count       int
}

// ConnectionFilter provides IP throttling, concurrent limit enforcement, and DoS mitigation.
type ConnectionFilter struct {
	mu           sync.Mutex
	opts         FilterOptions
	lastConnect  map[string]time.Time
	activeConns  map[string]int
	rateTrackers map[string]*rateTracker
}

var (
	defaultFilter     *ConnectionFilter
	defaultFilterOnce sync.Once
)

// DefaultFilter returns the global ConnectionFilter singleton.
func DefaultFilter() *ConnectionFilter {
	defaultFilterOnce.Do(func() {
		defaultFilter = NewFilter(DefaultFilterOptions())
	})
	return defaultFilter
}

// NewFilter creates a new ConnectionFilter with the given options.
func NewFilter(opts FilterOptions) *ConnectionFilter {
	return &ConnectionFilter{
		opts:         opts,
		lastConnect:  make(map[string]time.Time),
		activeConns:  make(map[string]int),
		rateTrackers: make(map[string]*rateTracker),
	}
}

// ExtractIP extracts the host IP string from a net.Addr.
func ExtractIP(addr net.Addr) string {
	if addr == nil {
		return "127.0.0.1"
	}
	str := addr.String()
	host, _, err := net.SplitHostPort(str)
	if err != nil {
		return strings.TrimSpace(str)
	}
	return strings.TrimSpace(host)
}

// Allow evaluates an incoming connection attempt against throttle, rate limits, and concurrent caps.
func (f *ConnectionFilter) Allow(addr net.Addr) (bool, string) {
	ip := ExtractIP(addr)
	now := time.Now()

	f.mu.Lock()
	defer f.mu.Unlock()

	// 1. Connection Throttle Check (Paper connection-throttle)
	if f.opts.ConnectionThrottle > 0 {
		if last, ok := f.lastConnect[ip]; ok {
			elapsed := now.Sub(last)
			if elapsed < f.opts.ConnectionThrottle {
				return false, fmt.Sprintf("Connection throttled! Please wait %v before reconnecting.", (f.opts.ConnectionThrottle - elapsed).Round(time.Millisecond))
			}
		}
	}

	// 2. Rate Limit Window Check
	if f.opts.RateLimitPerSecond > 0 {
		tracker, ok := f.rateTrackers[ip]
		if !ok || now.Sub(tracker.windowStart) >= time.Second {
			tracker = &rateTracker{windowStart: now, count: 0}
			f.rateTrackers[ip] = tracker
		}
		tracker.count++
		if tracker.count > f.opts.RateLimitPerSecond {
			return false, "Connection rate limit exceeded! Too many connection attempts."
		}
	}

	// 3. Max Concurrent Connections Per IP Check
	if f.opts.MaxConnectionsPerIP > 0 {
		if current := f.activeConns[ip]; current >= f.opts.MaxConnectionsPerIP {
			return false, fmt.Sprintf("Too many active connections from your IP (maximum is %d)!", f.opts.MaxConnectionsPerIP)
		}
	}

	// Record success and increment active count
	f.lastConnect[ip] = now
	f.activeConns[ip]++
	return true, ""
}

// Release decrements the active concurrent connection count for the given IP address.
func (f *ConnectionFilter) Release(addr net.Addr) {
	ip := ExtractIP(addr)

	f.mu.Lock()
	defer f.mu.Unlock()

	if count, ok := f.activeConns[ip]; ok {
		if count <= 1 {
			delete(f.activeConns, ip)
		} else {
			f.activeConns[ip]--
		}
	}
}

// ActiveConnections returns the current active connections count for an IP.
func (f *ConnectionFilter) ActiveConnections(addr net.Addr) int {
	ip := ExtractIP(addr)

	f.mu.Lock()
	defer f.mu.Unlock()
	return f.activeConns[ip]
}

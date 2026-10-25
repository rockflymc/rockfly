package network

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

var (
	// ErrMaxDepthExceeded is returned when NBT structure nesting exceeds the safety depth limit.
	ErrMaxDepthExceeded = errors.New("nbt guard: maximum nesting depth exceeded (possible decompression bomb)")
	// ErrOversizedPayload is returned when a byte array, string, or payload exceeds size limits.
	ErrOversizedPayload = errors.New("nbt guard: oversized payload or byte array allocation limit exceeded")
	// ErrElementLimitExceeded is returned when an NBT compound or list contains too many elements.
	ErrElementLimitExceeded = errors.New("nbt guard: element count limit exceeded")
	// ErrPacketRateExceeded is returned when a client floods packets faster than allowable rate limits.
	ErrPacketRateExceeded = errors.New("packet throttler: packet rate limit exceeded")
)

// NBTGuardConfig holds security thresholds for NBT decoding and payload validation.
type NBTGuardConfig struct {
	// MaxDepth is the maximum allowable compound/list recursion depth. Default is 32.
	MaxDepth int
	// MaxPayloadSize is the maximum allowable raw payload size in bytes. Default is 2MB.
	MaxPayloadSize int
	// MaxStringLength is the maximum allowable individual string length. Default is 64KB.
	MaxStringLength int
	// MaxElements is the maximum allowable element count per compound or list. Default is 10,000.
	MaxElements int
	// MaxBookPages is the maximum allowable pages in a book. Default is 100.
	MaxBookPages int
	// MaxBookPageLength is the maximum characters per book page. Default is 1024.
	MaxBookPageLength int
	// MaxSignLineLength is the maximum characters per sign line. Default is 128.
	MaxSignLineLength int
}

// DefaultNBTGuardConfig returns production-hardened Paper-grade safety thresholds.
func DefaultNBTGuardConfig() NBTGuardConfig {
	return NBTGuardConfig{
		MaxDepth:          32,
		MaxPayloadSize:    2 * 1024 * 1024, // 2MB
		MaxStringLength:   64 * 1024,       // 64KB
		MaxElements:       10000,
		MaxBookPages:      100,
		MaxBookPageLength: 1024,
		MaxSignLineLength: 128,
	}
}

// NBTGuard validates incoming NBT payloads and raw byte arrays against decompression bombs,
// memory exhaustion exploits, and nested compound recursion attacks.
type NBTGuard struct {
	conf NBTGuardConfig
}

// NewNBTGuard creates a new NBTGuard with the specified configuration.
func NewNBTGuard(conf NBTGuardConfig) *NBTGuard {
	return &NBTGuard{conf: conf}
}

// ValidateRawData verifies that the raw byte buffer is within permissible size bounds.
func (g *NBTGuard) ValidateRawData(data []byte) error {
	if len(data) > g.conf.MaxPayloadSize {
		return fmt.Errorf("%w: payload size %d exceeds limit %d", ErrOversizedPayload, len(data), g.conf.MaxPayloadSize)
	}
	return nil
}

// DecodeSafe safely unmarshals an NBT payload enforcing depth, element, and allocation bounds.
func (g *NBTGuard) DecodeSafe(data []byte, encoding nbt.Encoding) (map[string]any, error) {
	if err := g.ValidateRawData(data); err != nil {
		return nil, err
	}

	var m map[string]any
	decoder := nbt.NewDecoderWithEncoding(bytes.NewBuffer(data), encoding)
	if err := decoder.Decode(&m); err != nil {
		return nil, fmt.Errorf("nbt decode error: %w", err)
	}

	if err := g.validateValue(m, 0); err != nil {
		return nil, err
	}

	return m, nil
}

// validateValue recursively traverses decoded NBT values ensuring depth and size constraints.
func (g *NBTGuard) validateValue(v any, depth int) error {
	if depth > g.conf.MaxDepth {
		return fmt.Errorf("%w: current depth %d > max %d", ErrMaxDepthExceeded, depth, g.conf.MaxDepth)
	}

	switch val := v.(type) {
	case map[string]any:
		if len(val) > g.conf.MaxElements {
			return fmt.Errorf("%w: map has %d elements (max %d)", ErrElementLimitExceeded, len(val), g.conf.MaxElements)
		}
		for k, child := range val {
			if len(k) > g.conf.MaxStringLength {
				return fmt.Errorf("%w: tag key length %d exceeds max %d", ErrOversizedPayload, len(k), g.conf.MaxStringLength)
			}
			if err := g.validateValue(child, depth+1); err != nil {
				return err
			}
		}
	case []any:
		if len(val) > g.conf.MaxElements {
			return fmt.Errorf("%w: list has %d elements (max %d)", ErrElementLimitExceeded, len(val), g.conf.MaxElements)
		}
		for _, child := range val {
			if err := g.validateValue(child, depth+1); err != nil {
				return err
			}
		}
	case string:
		if len(val) > g.conf.MaxStringLength {
			return fmt.Errorf("%w: string length %d exceeds max %d", ErrOversizedPayload, len(val), g.conf.MaxStringLength)
		}
	case []byte:
		if len(val) > g.conf.MaxPayloadSize {
			return fmt.Errorf("%w: byte array size %d exceeds max %d", ErrOversizedPayload, len(val), g.conf.MaxPayloadSize)
		}
	}

	return nil
}

// ValidateSignText validates text written to signs, rejecting lines exceeding MaxSignLineLength.
func (g *NBTGuard) ValidateSignText(lines []string) error {
	for i, line := range lines {
		if len(line) > g.conf.MaxSignLineLength {
			return fmt.Errorf("%w: sign line %d has %d chars (max %d)", ErrOversizedPayload, i, len(line), g.conf.MaxSignLineLength)
		}
	}
	return nil
}

// ValidateBookPages validates book & quill contents against page count and page length limits.
func (g *NBTGuard) ValidateBookPages(pages []string) error {
	if len(pages) > g.conf.MaxBookPages {
		return fmt.Errorf("%w: book has %d pages (max %d)", ErrOversizedPayload, len(pages), g.conf.MaxBookPages)
	}
	for i, page := range pages {
		if len(page) > g.conf.MaxBookPageLength {
			return fmt.Errorf("%w: book page %d has %d chars (max %d)", ErrOversizedPayload, i, len(page), g.conf.MaxBookPageLength)
		}
	}
	return nil
}

// PacketThrottler monitors incoming packet frequency per connection to mitigate packet floods and DoS.
type PacketThrottler struct {
	mu           sync.Mutex
	maxPerSecond int
	windowStart  time.Time
	count        int
	violations   int
}

// NewPacketThrottler creates a PacketThrottler with the given packet per second limit.
func NewPacketThrottler(maxPerSecond int) *PacketThrottler {
	if maxPerSecond <= 0 {
		maxPerSecond = 500
	}
	return &PacketThrottler{
		maxPerSecond: maxPerSecond,
		windowStart:  time.Now(),
	}
}

// Allow increments the incoming packet count and returns true if within allowable thresholds.
// If packet rate exceeds limit, returns false with an error.
func (t *PacketThrottler) Allow() (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if now.Sub(t.windowStart) >= time.Second {
		t.windowStart = now
		t.count = 1
		return true, nil
	}

	t.count++
	if t.count > t.maxPerSecond {
		t.violations++
		return false, fmt.Errorf("%w: received %d pkts/sec (max %d)", ErrPacketRateExceeded, t.count, t.maxPerSecond)
	}

	return true, nil
}

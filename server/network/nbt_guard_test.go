package network

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func TestNBTGuard_NormalPayload(t *testing.T) {
	guard := NewNBTGuard(DefaultNBTGuardConfig())

	data := map[string]any{
		"Name":   "Steve",
		"Health": float32(20.0),
		"Inventory": []any{
			map[string]any{"Slot": byte(0), "Count": byte(64), "Item": "minecraft:apple"},
		},
	}

	buf := new(bytes.Buffer)
	if err := nbt.NewEncoderWithEncoding(buf, nbt.LittleEndian).Encode(data); err != nil {
		t.Fatal(err)
	}

	decoded, err := guard.DecodeSafe(buf.Bytes(), nbt.LittleEndian)
	if err != nil {
		t.Fatalf("expected valid payload to decode, got: %v", err)
	}
	if decoded["Name"] != "Steve" {
		t.Fatalf("expected Name Steve, got %v", decoded["Name"])
	}
}

func TestNBTGuard_RecursionDepthBombRejected(t *testing.T) {
	guard := NewNBTGuard(DefaultNBTGuardConfig())

	// Build a 50-level deep recursively nested compound tag (max is 32)
	root := make(map[string]any)
	curr := root
	for i := 0; i < 50; i++ {
		next := make(map[string]any)
		curr["child"] = next
		curr = next
	}
	curr["payload"] = "bomb"

	buf := new(bytes.Buffer)
	if err := nbt.NewEncoderWithEncoding(buf, nbt.LittleEndian).Encode(root); err != nil {
		t.Fatal(err)
	}

	_, err := guard.DecodeSafe(buf.Bytes(), nbt.LittleEndian)
	if err == nil {
		t.Fatal("expected 50-depth NBT bomb to be rejected")
	}
	if !errors.Is(err, ErrMaxDepthExceeded) {
		t.Fatalf("expected ErrMaxDepthExceeded, got: %v", err)
	}
}

func TestNBTGuard_OversizedRawPayloadRejected(t *testing.T) {
	guard := NewNBTGuard(DefaultNBTGuardConfig())

	// 3MB payload exceeds 2MB limit
	oversized := make([]byte, 3*1024*1024)
	err := guard.ValidateRawData(oversized)
	if err == nil {
		t.Fatal("expected 3MB payload to be rejected")
	}
	if !errors.Is(err, ErrOversizedPayload) {
		t.Fatalf("expected ErrOversizedPayload, got: %v", err)
	}
}

func TestNBTGuard_SignAndBookLimits(t *testing.T) {
	guard := NewNBTGuard(DefaultNBTGuardConfig())

	// Valid sign
	validSign := []string{"Welcome to", "Rockfly", "Server", ""}
	if err := guard.ValidateSignText(validSign); err != nil {
		t.Fatalf("expected valid sign, got: %v", err)
	}

	// Oversized sign line (>128 chars)
	invalidSign := []string{strings.Repeat("A", 150)}
	if err := guard.ValidateSignText(invalidSign); err == nil {
		t.Fatal("expected oversized sign line to be rejected")
	}

	// Valid book
	validBook := []string{"Page 1 text", "Page 2 text"}
	if err := guard.ValidateBookPages(validBook); err != nil {
		t.Fatalf("expected valid book, got: %v", err)
	}

	// Oversized book (>100 pages)
	invalidPages := make([]string, 150)
	if err := guard.ValidateBookPages(invalidPages); err == nil {
		t.Fatal("expected 150-page book to be rejected")
	}
}

func TestPacketThrottler_RateLimiting(t *testing.T) {
	throttler := NewPacketThrottler(100)

	// Send 100 packets within rate limit
	for i := 0; i < 100; i++ {
		allowed, err := throttler.Allow()
		if !allowed || err != nil {
			t.Fatalf("packet %d within limit was rejected: %v", i, err)
		}
	}

	// 101st packet must be rejected
	allowed, err := throttler.Allow()
	if allowed || err == nil {
		t.Fatal("expected packet 101 exceeding 100 pkts/s to be rejected")
	}
	if !errors.Is(err, ErrPacketRateExceeded) {
		t.Fatalf("expected ErrPacketRateExceeded, got: %v", err)
	}
}

func BenchmarkPacketThrottler_Allow(b *testing.B) {
	throttler := NewPacketThrottler(1000000000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = throttler.Allow()
	}
}

func TestNBTGuard_NilAndZeroSafety(t *testing.T) {
	// 1. Nil guard checks
	var nilGuard *NBTGuard
	if err := nilGuard.ValidateRawData([]byte("test")); err != nil {
		t.Fatalf("expected nil error on nil guard ValidateRawData, got: %v", err)
	}
	m, err := nilGuard.DecodeSafe(nil, nbt.LittleEndian)
	if m != nil || err != nil {
		t.Fatalf("expected nil result on nil guard DecodeSafe")
	}
	if err := nilGuard.ValidateSignText(nil); err != nil {
		t.Fatalf("expected nil error on nil guard ValidateSignText")
	}
	if err := nilGuard.ValidateBookPages(nil); err != nil {
		t.Fatalf("expected nil error on nil guard ValidateBookPages")
	}

	// 2. Zero-valued guard
	var zeroGuard NBTGuard
	if err := zeroGuard.ValidateRawData(make([]byte, 100)); err != nil {
		t.Fatalf("expected zeroGuard to use fallback default limits, got: %v", err)
	}

	// 3. Nil throttler
	var nilThrottler *PacketThrottler
	allowed, err := nilThrottler.Allow()
	if !allowed || err != nil {
		t.Fatalf("expected true, nil on nil throttler Allow")
	}

	// 4. Zero-valued throttler
	var zeroThrottler PacketThrottler
	allowed, err = zeroThrottler.Allow()
	if !allowed || err != nil {
		t.Fatalf("expected true, nil on zero throttler Allow")
	}
}

package session

import (
	"bytes"
	"testing"
)

func TestBufferPool_Tiers(t *testing.T) {
	pool := NewBufferPool()

	// Small tier (<= 1KB)
	s := pool.Get(512)
	if len(s) != 512 || cap(s) < SmallBufferSize {
		t.Fatalf("expected small buffer with len 512 and cap >= %d, got len=%d, cap=%d", SmallBufferSize, len(s), cap(s))
	}
	s[0] = 0xAA
	pool.Put(s)

	// Medium tier (<= 64KB)
	m := pool.Get(32 * 1024)
	if len(m) != 32*1024 || cap(m) < MediumBufferSize {
		t.Fatalf("expected medium buffer with cap >= %d", MediumBufferSize)
	}
	pool.Put(m)

	// Large tier (<= 512KB)
	l := pool.Get(256 * 1024)
	if len(l) != 256*1024 || cap(l) < LargeBufferSize {
		t.Fatalf("expected large buffer with cap >= %d", LargeBufferSize)
	}
	pool.Put(l)
}

func TestBufferPool_BufferAcquireRelease(t *testing.T) {
	buf := AcquireBuffer()
	if buf.Len() != 0 {
		t.Fatalf("expected empty buffer")
	}

	buf.WriteString("Rockfly PaperMC Performance")
	if buf.Len() == 0 {
		t.Fatalf("expected non-empty buffer after write")
	}

	ReleaseBuffer(buf)

	buf2 := AcquireBuffer()
	if buf2.Len() != 0 {
		t.Fatalf("expected recycled buffer to be cleanly reset to len 0")
	}
	ReleaseBuffer(buf2)
}

func BenchmarkBufferPool_AcquireReleaseBuffer(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := AcquireBuffer()
		buf.Write([]byte("benchmark packet serialization payload"))
		ReleaseBuffer(buf)
	}
}

func BenchmarkBufferPool_WithoutPool_HeapAlloc(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(nil)
		buf.Write([]byte("benchmark packet serialization payload"))
		_ = buf.Bytes()
	}
}

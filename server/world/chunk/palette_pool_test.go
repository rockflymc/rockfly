package chunk

import (
	"testing"
)

func TestPalettePool_Indices(t *testing.T) {
	pool := NewPalettePool()

	indices := pool.GetIndices(100)
	if len(indices) != 100 {
		t.Fatalf("expected 100 length, got %d", len(indices))
	}
	if cap(indices) < StandardPaletteCapacity {
		t.Fatalf("expected cap >= %d, got %d", StandardPaletteCapacity, cap(indices))
	}

	indices[0] = 42
	pool.PutIndices(indices)

	// Second acquire should be clean (zeroed)
	indices2 := pool.GetIndices(100)
	if indices2[0] != 0 {
		t.Fatalf("expected recycled palette slice to be cleared, got %d", indices2[0])
	}
	pool.PutIndices(indices2)
}

func TestPalettePool_Bytes(t *testing.T) {
	pool := NewPalettePool()

	b := pool.GetBytes(256)
	if len(b) != 256 {
		t.Fatalf("expected 256 length, got %d", len(b))
	}
	b[0] = 0xFF
	pool.PutBytes(b)

	b2 := pool.GetBytes(256)
	if b2[0] != 0 {
		t.Fatalf("expected recycled byte slice to be cleared, got %d", b2[0])
	}
	pool.PutBytes(b2)
}

func TestPalettePool_NilAndZeroSafety(t *testing.T) {
	var nilPool *PalettePool
	if nilPool.GetIndices(10) == nil || len(nilPool.GetIndices(10)) != 10 {
		t.Fatalf("expected fallback allocation from nil PalettePool")
	}
	if nilPool.GetIndices(-5) != nil || nilPool.GetBytes(-5) != nil {
		t.Fatalf("expected nil from negative capacity")
	}
	nilPool.PutIndices([]uint32{1, 2, 3})
	nilPool.PutBytes([]byte{1, 2, 3})

	zeroPool := &PalettePool{}
	ind := zeroPool.GetIndices(16)
	if len(ind) != 16 {
		t.Fatalf("expected slice from zero-value pool")
	}
	zeroPool.PutIndices(ind)
	bs := zeroPool.GetBytes(32)
	if len(bs) != 32 {
		t.Fatalf("expected bytes from zero-value pool")
	}
	zeroPool.PutBytes(bs)
}

func BenchmarkPalettePool_AcquireRelease(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		indices := AcquirePaletteIndices(4096)
		indices[0] = 123
		ReleasePaletteIndices(indices)
	}
}

func BenchmarkPalettePool_WithoutPool(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		indices := make([]uint32, 4096)
		indices[0] = 123
		_ = indices
	}
}

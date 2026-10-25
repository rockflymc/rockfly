package world

import (
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

func TestChunkCache_LRUEviction(t *testing.T) {
	cache := NewChunkCache(3)
	r := cube.Range{-64, 319}
	br := DefaultBlockRegistry
	br.Finalize()

	c1 := &Column{Chunk: chunk.New(br, r)}
	c2 := &Column{Chunk: chunk.New(br, r)}
	c3 := &Column{Chunk: chunk.New(br, r)}
	c4 := &Column{Chunk: chunk.New(br, r)}

	p1 := ChunkPos{1, 0}
	p2 := ChunkPos{2, 0}
	p3 := ChunkPos{3, 0}
	p4 := ChunkPos{4, 0}

	cache.Put(p1, c1)
	cache.Put(p2, c2)
	cache.Put(p3, c3)

	if cache.Len() != 3 {
		t.Fatalf("expected len 3, got %d", cache.Len())
	}

	// Access p1 so it becomes most recently used
	if _, ok := cache.Get(p1); !ok {
		t.Fatal("expected p1 to be in cache")
	}

	// Insert p4, causing LRU (which is p2, since p1 was accessed) to be evicted
	cache.Put(p4, c4)

	if cache.Len() != 3 {
		t.Fatalf("expected len 3 after eviction, got %d", cache.Len())
	}

	if _, ok := cache.Get(p2); ok {
		t.Fatal("expected p2 to be evicted from cache")
	}
	if _, ok := cache.Get(p1); !ok {
		t.Fatal("expected p1 to remain in cache")
	}
	if _, ok := cache.Get(p3); !ok {
		t.Fatal("expected p3 to remain in cache")
	}
	if _, ok := cache.Get(p4); !ok {
		t.Fatal("expected p4 to remain in cache")
	}
}

func TestChunkCache_ConcurrentAccess(t *testing.T) {
	cache := NewChunkCache(100)
	r := cube.Range{-64, 319}
	br := DefaultBlockRegistry
	br.Finalize()

	var wg sync.WaitGroup
	const goroutines = 10
	const opsPerRoutine = 100
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < opsPerRoutine; i++ {
				pos := ChunkPos{int32(gid*100 + i), 0}
				col := &Column{Chunk: chunk.New(br, r)}
				cache.Put(pos, col)
				_, _ = cache.Get(pos)
				if i%10 == 0 {
					cache.Remove(pos)
				}
			}
		}(g)
	}

	wg.Wait()
}

func BenchmarkChunkCache_GetPut(b *testing.B) {
	cache := NewChunkCache(512)
	r := cube.Range{-64, 319}
	br := DefaultBlockRegistry
	br.Finalize()
	col := &Column{Chunk: chunk.New(br, r)}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pos := ChunkPos{int32(i & 0x1ff), 0}
		cache.Put(pos, col)
		_, _ = cache.Get(pos)
	}
}

func TestChunkCache_NilAndZeroSafety(t *testing.T) {
	var nilCache *ChunkCache
	if _, ok := nilCache.Get(ChunkPos{0, 0}); ok {
		t.Fatal("expected false on nil Get")
	}
	nilCache.Put(ChunkPos{0, 0}, nil)
	nilCache.Remove(ChunkPos{0, 0})
	nilCache.Clear()
	if nilCache.Len() != 0 {
		t.Fatal("expected 0 on nil Len")
	}

	var zeroCache ChunkCache
	if _, ok := zeroCache.Get(ChunkPos{0, 0}); ok {
		t.Fatal("expected false on zero Get")
	}
	br := DefaultBlockRegistry
	br.Finalize()
	r := cube.Range{-64, 319}
	zeroCache.Put(ChunkPos{1, 1}, &Column{Chunk: chunk.New(br, r)})
	if zeroCache.Len() != 1 {
		t.Fatalf("expected len 1, got %d", zeroCache.Len())
	}
	if _, ok := zeroCache.Get(ChunkPos{1, 1}); !ok {
		t.Fatal("expected ok on zeroCache Get after Put")
	}
	zeroCache.Clear()
	if zeroCache.Len() != 0 {
		t.Fatal("expected 0 after Clear")
	}
}

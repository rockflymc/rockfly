package mcdb_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
)

func TestAsyncWriter_DoubleBufferedFlush(t *testing.T) {
	dir, err := os.MkdirTemp("", "mcdb_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := mcdb.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	writer := mcdb.NewAsyncWriter(db)
	defer writer.Close()

	r := cube.Range{-64, 319}
	br := world.DefaultBlockRegistry

	// Write 100 chunks across different positions
	const chunkCount = 100
	for i := 0; i < chunkCount; i++ {
		ch := chunk.New(br, r)
		col := &chunk.Column{Chunk: ch}
		pos := world.ChunkPos{int32(i), int32(i * 2)}

		if err := writer.QueueWrite(pos, world.Overworld, col); err != nil {
			t.Fatalf("failed to queue write %d: %v", i, err)
		}
	}

	// Flush all writes to LevelDB
	if err := writer.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	// Verify that all 100 chunks were persisted and can be loaded
	for i := 0; i < chunkCount; i++ {
		pos := world.ChunkPos{int32(i), int32(i * 2)}
		loaded, err := db.LoadColumn(pos, world.Overworld)
		if err != nil {
			t.Fatalf("failed to load chunk %d at %v: %v", i, pos, err)
		}
		if loaded == nil || loaded.Chunk == nil {
			t.Fatalf("loaded chunk %d is nil", i)
		}
	}
}

func TestAsyncReader_ConcurrentReads(t *testing.T) {
	dir, err := os.MkdirTemp("", "mcdb_test_reader_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := mcdb.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	r := cube.Range{-64, 319}
	br := world.DefaultBlockRegistry

	// Pre-populate 20 chunks
	const count = 20
	for i := 0; i < count; i++ {
		ch := chunk.New(br, r)
		col := &chunk.Column{Chunk: ch}
		pos := world.ChunkPos{int32(i), 0}
		if err := db.StoreColumn(pos, world.Overworld, col); err != nil {
			t.Fatal(err)
		}
	}

	reader := mcdb.NewAsyncReader(db, 4)
	defer reader.Close()

	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		pos := world.ChunkPos{int32(i), 0}
		reader.QueueRead(pos, world.Overworld, func(col *chunk.Column, err error) {
			defer wg.Done()
			if err != nil {
				t.Errorf("read failed for %v: %v", pos, err)
			}
			if col == nil || col.Chunk == nil {
				t.Errorf("chunk at %v was nil", pos)
			}
		})
	}

	wg.Wait()
}

func BenchmarkAsyncWriter_QueueWrite(b *testing.B) {
	dir := filepath.Join(os.TempDir(), "mcdb_bench")
	_ = os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	db, err := mcdb.Open(dir)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	writer := mcdb.NewAsyncWriter(db)
	defer writer.Close()

	r := cube.Range{-64, 319}
	br := world.DefaultBlockRegistry
	ch := chunk.New(br, r)
	col := &chunk.Column{Chunk: ch}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pos := world.ChunkPos{int32(i & 0xff), int32((i >> 8) & 0xff)}
		_ = writer.QueueWrite(pos, world.Overworld, col)
	}
}

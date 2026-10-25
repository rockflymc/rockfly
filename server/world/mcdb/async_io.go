package mcdb

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb"
)

// AsyncWriter implements double-buffered asynchronous batch writing for LevelDB.
// Dirty chunk columns are serialized and buffered off the main world tick thread,
// eliminating disk I/O stalls during chunk saving and player movement.
type AsyncWriter struct {
	db     *DB
	mu     sync.Mutex
	cond   *sync.Cond
	closed atomic.Bool

	activeBatch   *leveldb.Batch
	flushingBatch *leveldb.Batch

	activeCount   int
	flushingCount int

	flushTrigger chan struct{}
	done         chan struct{}
	flushDone    chan struct{}
}

// NewAsyncWriter creates a new AsyncWriter bound to the specified DB.
func NewAsyncWriter(db *DB) *AsyncWriter {
	w := &AsyncWriter{
		db:            db,
		activeBatch:   new(leveldb.Batch),
		flushingBatch: new(leveldb.Batch),
		flushTrigger:  make(chan struct{}, 1),
		done:          make(chan struct{}),
		flushDone:     make(chan struct{}, 1),
	}
	w.cond = sync.NewCond(&w.mu)
	go w.flushLoop()
	return w
}

// QueueWrite serializes the given column and adds its entries to the active write batch.
func (w *AsyncWriter) QueueWrite(pos world.ChunkPos, dim world.Dimension, col *chunk.Column) error {
	if w.closed.Load() {
		return fmt.Errorf("async writer is closed")
	}

	k := dbKey{pos: pos, dim: dim}
	// Encode chunk data outside the lock to maximize concurrency
	data := chunk.Encode(col.Chunk, chunk.DiskEncoding)

	w.mu.Lock()
	defer w.mu.Unlock()

	w.db.storeVersion(w.activeBatch, k, chunkVersion)
	w.db.storeBiomes(w.activeBatch, k, data.Biomes)
	w.db.storeSubChunks(w.activeBatch, k, data.SubChunks, col.Chunk.Range())
	w.db.storeFinalisation(w.activeBatch, k, finalisationPopulated)
	w.db.storeLegacyStates(w.activeBatch, k, col.Chunk.LegacyStates())
	w.db.storeEntities(w.activeBatch, k, col.Entities)
	w.db.storeBlockEntities(w.activeBatch, k, col.BlockEntities)
	w.db.storeScheduledUpdates(w.activeBatch, k, col.Tick, col.ScheduledBlocks)

	w.activeCount++

	if w.activeCount >= 64 {
		select {
		case w.flushTrigger <- struct{}{}:
		default:
		}
	}

	return nil
}

// flushLoop continuously drains and flushes batches to disk on a background goroutine.
func (w *AsyncWriter) flushLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	defer close(w.done)

	for {
		select {
		case <-w.flushTrigger:
			w.flushActive()
		case <-ticker.C:
			w.flushActive()
		}

		if w.closed.Load() {
			w.flushActive()
			return
		}
	}
}

// flushActive swaps activeBatch into flushingBatch and writes to disk.
func (w *AsyncWriter) flushActive() {
	w.mu.Lock()
	if w.activeCount == 0 {
		w.mu.Unlock()
		return
	}

	// Swap active and flushing buffers.
	w.flushingBatch, w.activeBatch = w.activeBatch, w.flushingBatch
	w.flushingCount = w.activeCount
	w.activeBatch.Reset()
	w.activeCount = 0

	batch := w.flushingBatch
	w.mu.Unlock()

	// Write batch to LevelDB synchronously on this background worker thread.
	if err := w.db.ldb.Write(batch, nil); err != nil {
		w.db.conf.Log.Error("async writer: write batch failed: " + err.Error())
	}

	w.mu.Lock()
	w.flushingBatch.Reset()
	w.flushingCount = 0
	w.cond.Broadcast()
	w.mu.Unlock()

	select {
	case w.flushDone <- struct{}{}:
	default:
	}
}

// Flush synchronously flushes all queued writes to disk and blocks until complete.
func (w *AsyncWriter) Flush() error {
	w.mu.Lock()
	for w.activeCount > 0 || w.flushingCount > 0 {
		select {
		case w.flushTrigger <- struct{}{}:
		default:
		}
		w.cond.Wait()
	}
	w.mu.Unlock()
	return nil
}

// Close gracefully closes the writer after flushing all pending data.
func (w *AsyncWriter) Close() error {
	if w.closed.CompareAndSwap(false, true) {
		select {
		case w.flushTrigger <- struct{}{}:
		default:
		}
		<-w.done
	}
	return nil
}

// ReadTask represents a pending asynchronous chunk read task.
type ReadTask struct {
	Pos      world.ChunkPos
	Dim      world.Dimension
	Callback func(col *chunk.Column, err error)
}

// AsyncReader manages background worker goroutines to fetch raw chunk bytes and decode
// NBT and subchunks off the main world tick loop.
type AsyncReader struct {
	db      *DB
	tasks   chan ReadTask
	wg      sync.WaitGroup
	closed  atomic.Bool
	workers int
}

// NewAsyncReader creates a new AsyncReader with the given number of background read workers.
func NewAsyncReader(db *DB, workers int) *AsyncReader {
	if workers <= 0 {
		workers = 4
	}
	r := &AsyncReader{
		db:      db,
		tasks:   make(chan ReadTask, 2048),
		workers: workers,
	}
	for i := 0; i < workers; i++ {
		r.wg.Add(1)
		go r.workerLoop()
	}
	return r
}

// workerLoop executes chunk reading tasks on worker goroutines.
func (r *AsyncReader) workerLoop() {
	defer r.wg.Done()
	for task := range r.tasks {
		col, err := r.db.LoadColumn(task.Pos, task.Dim)
		if task.Callback != nil {
			task.Callback(col, err)
		}
	}
}

// QueueRead submits an asynchronous chunk read request to the background worker pool.
func (r *AsyncReader) QueueRead(pos world.ChunkPos, dim world.Dimension, callback func(*chunk.Column, error)) bool {
	if r.closed.Load() {
		return false
	}
	task := ReadTask{Pos: pos, Dim: dim, Callback: callback}
	select {
	case r.tasks <- task:
		return true
	default:
		// Queue full: perform load synchronously as fallback.
		col, err := r.db.LoadColumn(pos, dim)
		if callback != nil {
			callback(col, err)
		}
		return true
	}
}

// Close stops all reader worker goroutines.
func (r *AsyncReader) Close() {
	if r.closed.CompareAndSwap(false, true) {
		close(r.tasks)
		r.wg.Wait()
	}
}

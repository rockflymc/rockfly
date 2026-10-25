package starlight

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// TaskType describes the operation to be performed by the lighting engine.
type TaskType uint8

const (
	TaskFill TaskType = iota
	TaskSpread
	TaskUpdate
)

// LightTask represents an asynchronous unit of lighting calculation.
type LightTask struct {
	Type   TaskType
	Action func()
	OnDone func()
}

// Engine manages background worker goroutines that perform bit-packed lighting computations
// off the main tick thread.
type Engine struct {
	workers int
	tasks   chan LightTask
	wg      sync.WaitGroup
	mu      sync.RWMutex
	closed  atomic.Bool
}

// NewEngine creates a new Starlight lighting engine with the specified worker pool size.
// If workers <= 0, defaults to runtime.NumCPU() (minimum 2).
func NewEngine(workers int) *Engine {
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers < 2 {
			workers = 2
		}
	}

	e := &Engine{
		workers: workers,
		tasks:   make(chan LightTask, 1024),
	}

	for i := 0; i < workers; i++ {
		e.wg.Add(1)
		go e.workerLoop()
	}

	return e
}

// workerLoop is the execution loop for each background lighting worker.
func (e *Engine) workerLoop() {
	defer e.wg.Done()
	for task := range e.tasks {
		e.runTask(task)
	}
}

func (e *Engine) runTask(task LightTask) {
	defer func() {
		_ = recover()
	}()
	if task.Action != nil {
		task.Action()
	}
	if task.OnDone != nil {
		task.OnDone()
	}
}

// Submit queues a lighting task to be processed asynchronously by worker goroutines.
// Returns false if the engine has been shut down.
func (e *Engine) Submit(task LightTask) bool {
	if e == nil {
		e.runTask(task)
		return false
	}

	e.mu.RLock()
	if e.closed.Load() {
		e.mu.RUnlock()
		return false
	}

	select {
	case e.tasks <- task:
		e.mu.RUnlock()
		return true
	default:
		e.mu.RUnlock()
		// Task queue full: execute synchronously as graceful fallback to avoid dropping light updates.
		e.runTask(task)
		return true
	}
}

// Close gracefully stops all worker goroutines after draining queued tasks.
func (e *Engine) Close() {
	if e == nil {
		return
	}
	if e.closed.CompareAndSwap(false, true) {
		e.mu.Lock()
		close(e.tasks)
		e.mu.Unlock()
		e.wg.Wait()
	}
}

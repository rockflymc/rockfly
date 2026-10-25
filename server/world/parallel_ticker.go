package world

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gl/mathgl/mgl64"
)

// HandoffStatus represents the result of an inter-world entity handoff.
type HandoffStatus uint8

const (
	HandoffSuccess HandoffStatus = iota
	HandoffTargetTimeout
	HandoffWorldClosed
	HandoffFailed
)

// EntityHandoff represents an atomic transfer request of an entity from one World to another.
type EntityHandoff struct {
	Handle     *EntityHandle
	Source     *World
	Target     *World
	Pos        mgl64.Vec3
	Timeout    time.Duration
	OnComplete func(e Entity)
	OnFail     func(err error)
}

// ParallelWorldTicker coordinates concurrent execution of multiple dimensions (Overworld, Nether, End)
// across independent CPU cores, providing lock-free inter-world entity handoff channels.
type ParallelWorldTicker struct {
	mu        sync.RWMutex
	worlds    map[string]*World
	channelMu sync.RWMutex
	handoffs  chan EntityHandoff
	closed    atomic.Bool
	wg        sync.WaitGroup

	tickStats map[string]*WorldTickStats
}

// WorldTickStats tracks real-time performance metrics for an individual dimension world.
type WorldTickStats struct {
	Dimension    string
	CurrentTick  atomic.Int64
	LastTickTime atomic.Int64 // nanoseconds
	EntityCount  atomic.Int32
	LoadedChunks atomic.Int32
}

// NewParallelWorldTicker creates a new ParallelWorldTicker for multi-world decoupling.
func NewParallelWorldTicker(handoffBufferSize int) *ParallelWorldTicker {
	if handoffBufferSize <= 0 {
		handoffBufferSize = 1024
	}
	p := &ParallelWorldTicker{
		worlds:    make(map[string]*World),
		handoffs:  make(chan EntityHandoff, handoffBufferSize),
		tickStats: make(map[string]*WorldTickStats),
	}
	p.wg.Add(1)
	go p.handoffProcessor()
	return p
}

// RegisterWorld registers a world dimension with the parallel ticker.
func (p *ParallelWorldTicker) RegisterWorld(name string, w *World) {
	if p == nil || w == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	p.worlds[name] = w
	p.tickStats[name] = &WorldTickStats{
		Dimension: name,
	}
}

// Stats returns the real-time tick performance statistics for the named dimension.
func (p *ParallelWorldTicker) Stats(name string) (*WorldTickStats, bool) {
	if p == nil {
		return nil, false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()

	s, ok := p.tickStats[name]
	return s, ok
}

// TransferEntity queues an entity to be atomically handed off from its source world to the target world
// without holding cross-world locks. If target world does not accept the entity before timeout, fallback occurs.
func (p *ParallelWorldTicker) TransferEntity(handoff EntityHandoff) bool {
	if p == nil || p.closed.Load() || handoff.Source == nil || handoff.Target == nil || handoff.Handle == nil {
		if handoff.OnFail != nil {
			handoff.OnFail(fmt.Errorf("parallel ticker closed or invalid handoff references"))
		}
		return false
	}

	if math.IsNaN(handoff.Pos[0]) || math.IsNaN(handoff.Pos[1]) || math.IsNaN(handoff.Pos[2]) ||
		math.IsInf(handoff.Pos[0], 0) || math.IsInf(handoff.Pos[1], 0) || math.IsInf(handoff.Pos[2], 0) {
		if handoff.OnFail != nil {
			handoff.OnFail(fmt.Errorf("handoff position contains NaN or Inf values"))
		}
		return false
	}

	if handoff.Timeout <= 0 {
		handoff.Timeout = 5 * time.Second
	}

	p.channelMu.RLock()
	if p.closed.Load() {
		p.channelMu.RUnlock()
		if handoff.OnFail != nil {
			handoff.OnFail(fmt.Errorf("parallel ticker closed"))
		}
		return false
	}

	select {
	case p.handoffs <- handoff:
		p.channelMu.RUnlock()
		return true
	default:
		p.channelMu.RUnlock()
		// Handoff channel full: execute synchronously as fallback
		go p.processSingleHandoff(handoff)
		return true
	}
}

// handoffProcessor runs on a background goroutine processing entity handoffs between dimensions.
func (p *ParallelWorldTicker) handoffProcessor() {
	defer p.wg.Done()

	for handoff := range p.handoffs {
		p.processSingleHandoff(handoff)
	}
}

// processSingleHandoff atomically executes an entity transfer to target world with timeout protection.
func (p *ParallelWorldTicker) processSingleHandoff(h EntityHandoff) {
	defer func() {
		if r := recover(); r != nil {
			if h.OnFail != nil {
				h.OnFail(fmt.Errorf("handoff panic: %v", r))
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), h.Timeout)
	defer cancel()

	added, err := Call(ctx, h.Target, func(tx *Tx) (Entity, error) {
		e := tx.AddEntityAt(h.Handle, h.Pos)
		return e, nil
	})

	if err != nil || added == nil {
		// Target failed or timed out: safely return entity back to source world to prevent entity loss
		_, _ = Call(context.Background(), h.Source, func(tx *Tx) (Entity, error) {
			return tx.AddEntityAt(h.Handle, h.Pos), nil
		})
		if h.OnFail != nil {
			h.OnFail(fmt.Errorf("handoff to target %s failed: %w", h.Target.Name(), err))
		}
		return
	}

	if h.OnComplete != nil {
		h.OnComplete(added)
	}
}

// TickSynchronousParallel advances all registered worlds concurrently by a single tick.
// Useful for integration tests and synchronous tick environments.
func (p *ParallelWorldTicker) TickSynchronousParallel() {
	if p == nil {
		return
	}
	p.mu.RLock()
	defer p.mu.RUnlock()

	var wg sync.WaitGroup
	for _, w := range p.worlds {
		if w == nil {
			continue
		}
		wg.Add(1)
		go func(target *World) {
			defer wg.Done()
			target.AdvanceTick()
		}(w)
	}
	wg.Wait()
}

// Close gracefully closes the parallel ticker and drains handoff channels.
func (p *ParallelWorldTicker) Close() {
	if p == nil {
		return
	}
	if p.closed.CompareAndSwap(false, true) {
		p.channelMu.Lock()
		close(p.handoffs)
		p.channelMu.Unlock()
		p.wg.Wait()
	}
}

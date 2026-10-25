package event

import (
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
)

// TypedHandlerFunc represents a zero-allocation, strongly typed listener callback.
type TypedHandlerFunc[E any] func(event *E)

type typedListener[E any] struct {
	id              uint64
	priority        Priority
	ignoreCancelled bool
	invoke          TypedHandlerFunc[E]
}

// TypedSubscription represents an active typed event listener subscription.
type TypedSubscription interface {
	Unsubscribe()
}

type typedSubscriptionImpl[E any] struct {
	bus       *TypedBus[E]
	id        uint64
	cancelled atomic.Bool
}

func (s *typedSubscriptionImpl[E]) Unsubscribe() {
	if s.cancelled.CompareAndSwap(false, true) {
		s.bus.unsubscribe(s.id)
	}
}

// TypedBus provides zero-reflection, zero-allocation generic event dispatching
// specifically designed for high-frequency game events (movement, ticking, combat, block mutation).
type TypedBus[E any] struct {
	mu        sync.Mutex
	listeners []*typedListener[E]
	cached    atomic.Pointer[[]*typedListener[E]]
	nextID    atomic.Uint64
}

// NewTypedBus creates a new TypedBus for event type E.
func NewTypedBus[E any]() *TypedBus[E] {
	b := &TypedBus[E]{}
	b.rebuildCache()
	return b
}

func (b *TypedBus[E]) rebuildCache() {
	sorted := make([]*typedListener[E], len(b.listeners))
	copy(sorted, b.listeners)

	slices.SortStableFunc(sorted, func(a, b *typedListener[E]) int {
		if a.priority < b.priority {
			return -1
		}
		if a.priority > b.priority {
			return 1
		}
		return 0
	})

	b.cached.Store(&sorted)
}

func (b *TypedBus[E]) unsubscribe(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, l := range b.listeners {
		if l.id == id {
			b.listeners = append(b.listeners[:i], b.listeners[i+1:]...)
			b.rebuildCache()
			break
		}
	}
}

// Subscribe registers a typed callback with custom priority and cancellation handling.
func (b *TypedBus[E]) Subscribe(fn TypedHandlerFunc[E], opts ...ListenerOptions) TypedSubscription {
	opt := ListenerOptions{Priority: PriorityNormal}
	if len(opts) > 0 {
		opt = opts[0]
	}

	id := b.nextID.Add(1)
	l := &typedListener[E]{
		id:              id,
		priority:        opt.Priority,
		ignoreCancelled: opt.IgnoreCancelled,
		invoke:          fn,
	}

	b.mu.Lock()
	b.listeners = append(b.listeners, l)
	b.rebuildCache()
	b.mu.Unlock()

	return &typedSubscriptionImpl[E]{
		bus: b,
		id:  id,
	}
}

// On registers a typed callback with standard PriorityNormal.
func (b *TypedBus[E]) On(fn TypedHandlerFunc[E]) TypedSubscription {
	return b.Subscribe(fn)
}

// Publish dispatches the event pointer to all registered listeners with zero heap allocations
// and lock-free execution via atomic copy-on-write snapshot loading.
func (b *TypedBus[E]) Publish(event *E) {
	if event == nil {
		return
	}

	cached := b.cached.Load()
	if cached == nil || len(*cached) == 0 {
		return
	}

	// Check cancellation if event implements Cancellable
	cancellable, isCancellable := any(event).(Cancellable)
	resettable, isResettable := any(event).(ResettableCancellable)

	for _, l := range *cached {
		if isCancellable && l.ignoreCancelled && cancellable.Cancelled() {
			continue
		}

		var wasCancelled bool
		if l.priority == PriorityMonitor && isResettable {
			wasCancelled = cancellable.Cancelled()
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("recovered panic in typed event listener",
						"priority", l.priority.String(),
						"error", r,
					)
				}
			}()
			l.invoke(event)
		}()

		// Enforce invariant: Monitor priority listeners cannot cancel/uncancel an event
		if l.priority == PriorityMonitor && isResettable && cancellable.Cancelled() != wasCancelled {
			resettable.SetCancelled(wasCancelled)
		}
	}
}

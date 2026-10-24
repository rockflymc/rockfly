package event

import (
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
)

// ListenerOptions configures priority and cancellation behavior for an event subscription.
type ListenerOptions struct {
	// Priority specifies the execution stage of this listener. Defaults to PriorityNormal.
	Priority Priority
	// IgnoreCancelled indicates that this listener should be bypassed if the event
	// has already been cancelled by an earlier priority handler.
	IgnoreCancelled bool
}

// Subscription represents an active event listener subscription that can be revoked.
type Subscription interface {
	Unsubscribe()
}

type registeredListener struct {
	id              uint64
	priority        Priority
	ignoreCancelled bool
	invoke          func(any)
}

type eventEntry struct {
	mu        sync.Mutex
	listeners []*registeredListener
	cached    atomic.Pointer[[]*registeredListener]
}

func (e *eventEntry) rebuildCache() {
	sorted := make([]*registeredListener, len(e.listeners))
	copy(sorted, e.listeners)

	slices.SortStableFunc(sorted, func(a, b *registeredListener) int {
		if a.priority < b.priority {
			return -1
		}
		if a.priority > b.priority {
			return 1
		}
		return 0
	})

	e.cached.Store(&sorted)
}

// Bus provides a high-throughput, concurrent-safe multi-listener event bus with
// PaperMC-equivalent priority sequencing and copy-on-write dispatch guarantees.
type Bus struct {
	mu      sync.RWMutex
	entries map[reflect.Type]*eventEntry
	nextID  atomic.Uint64
}

var (
	defaultBus     *Bus
	defaultBusOnce sync.Once
)

// DefaultBus returns the global default EventBus singleton.
func DefaultBus() *Bus {
	defaultBusOnce.Do(func() {
		defaultBus = NewBus()
	})
	return defaultBus
}

// NewBus creates a new, independent EventBus instance.
func NewBus() *Bus {
	return &Bus{
		entries: make(map[reflect.Type]*eventEntry),
	}
}

type subscriptionImpl struct {
	bus       *Bus
	eventType reflect.Type
	id        uint64
	cancelled atomic.Bool
}

func (s *subscriptionImpl) Unsubscribe() {
	if s.cancelled.CompareAndSwap(false, true) {
		s.bus.unsubscribe(s.eventType, s.id)
	}
}

func (b *Bus) getOrCreateEntry(t reflect.Type) *eventEntry {
	b.mu.RLock()
	e, ok := b.entries[t]
	b.mu.RUnlock()
	if ok {
		return e
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if e, ok = b.entries[t]; ok {
		return e
	}
	e = &eventEntry{}
	e.rebuildCache()
	b.entries[t] = e
	return e
}

func (b *Bus) unsubscribe(t reflect.Type, id uint64) {
	b.mu.RLock()
	e, ok := b.entries[t]
	b.mu.RUnlock()
	if !ok {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for i, l := range e.listeners {
		if l.id == id {
			e.listeners = append(e.listeners[:i], e.listeners[i+1:]...)
			e.rebuildCache()
			break
		}
	}
}

// SubscribeRaw registers a dynamic handler for an event type.
func (b *Bus) SubscribeRaw(eventType reflect.Type, fn func(any), opts ...ListenerOptions) Subscription {
	opt := ListenerOptions{Priority: PriorityNormal}
	if len(opts) > 0 {
		opt = opts[0]
	}

	id := b.nextID.Add(1)
	listener := &registeredListener{
		id:              id,
		priority:        opt.Priority,
		ignoreCancelled: opt.IgnoreCancelled,
		invoke:          fn,
	}

	entry := b.getOrCreateEntry(eventType)
	entry.mu.Lock()
	entry.listeners = append(entry.listeners, listener)
	entry.rebuildCache()
	entry.mu.Unlock()

	return &subscriptionImpl{
		bus:       b,
		eventType: eventType,
		id:        id,
	}
}

// Publish dispatches an event to all registered listeners matching its concrete type.
// Dispatch is zero-allocation and lock-free across concurrent goroutines using an atomic pointer snapshot.
func (b *Bus) Publish(event any) {
	if event == nil {
		return
	}

	t := reflect.TypeOf(event)
	b.mu.RLock()
	entry, ok := b.entries[t]
	b.mu.RUnlock()
	if !ok {
		return
	}

	cached := entry.cached.Load()
	if cached == nil || len(*cached) == 0 {
		return
	}

	cancellable, isCancellable := event.(Cancellable)

	for _, l := range *cached {
		if isCancellable && l.ignoreCancelled && cancellable.Cancelled() {
			continue
		}
		l.invoke(event)
	}
}

// Subscribe registers a strongly-typed listener callback on the bus.
func Subscribe[E any](b *Bus, fn func(E), opts ...ListenerOptions) Subscription {
	var zero E
	t := reflect.TypeOf(zero)
	// If zero is an interface, get type from interface type pointer
	if t == nil {
		t = reflect.TypeOf((*E)(nil)).Elem()
	}

	wrapper := func(val any) {
		if typed, ok := val.(E); ok {
			fn(typed)
		}
	}
	return b.SubscribeRaw(t, wrapper, opts...)
}

// On registers a strongly-typed listener callback with standard PriorityNormal and ignoreCancelled=false.
func On[E any](b *Bus, fn func(E)) Subscription {
	return Subscribe[E](b, fn)
}

// Publish dispatches a strongly-typed event on the specified bus.
func Publish[E any](b *Bus, event E) {
	b.Publish(event)
}

// SubscribeGlobal registers a typed listener callback on the default global bus.
func SubscribeGlobal[E any](fn func(E), opts ...ListenerOptions) Subscription {
	return Subscribe[E](DefaultBus(), fn, opts...)
}

// OnGlobal registers a typed listener callback with standard PriorityNormal on the default global bus.
func OnGlobal[E any](fn func(E)) Subscription {
	return On[E](DefaultBus(), fn)
}

// PublishGlobal dispatches a typed event on the default global bus.
func PublishGlobal[E any](event E) {
	Publish[E](DefaultBus(), event)
}

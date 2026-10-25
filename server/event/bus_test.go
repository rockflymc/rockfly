package event

import (
	"sync"
	"sync/atomic"
	"testing"
)

type customTestEvent struct {
	message   string
	cancelled bool
}

func (e *customTestEvent) Cancelled() bool { return e.cancelled }
func (e *customTestEvent) Cancel()         { e.cancelled = true }

func TestEventPriorityOrder(t *testing.T) {
	bus := NewBus()
	var order []string

	// Register in random order
	Subscribe(bus, func(e *customTestEvent) {
		order = append(order, "MONITOR")
	}, ListenerOptions{Priority: PriorityMonitor})

	Subscribe(bus, func(e *customTestEvent) {
		order = append(order, "LOWEST")
	}, ListenerOptions{Priority: PriorityLowest})

	Subscribe(bus, func(e *customTestEvent) {
		order = append(order, "NORMAL")
	}, ListenerOptions{Priority: PriorityNormal})

	Subscribe(bus, func(e *customTestEvent) {
		order = append(order, "HIGHEST")
	}, ListenerOptions{Priority: PriorityHighest})

	Subscribe(bus, func(e *customTestEvent) {
		order = append(order, "LOW")
	}, ListenerOptions{Priority: PriorityLow})

	Subscribe(bus, func(e *customTestEvent) {
		order = append(order, "HIGH")
	}, ListenerOptions{Priority: PriorityHigh})

	ev := &customTestEvent{message: "test"}
	Publish(bus, ev)

	expected := []string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d listeners to run, got %d", len(expected), len(order))
	}
	for i, name := range expected {
		if order[i] != name {
			t.Errorf("step %d: expected %s, got %s", i, name, order[i])
		}
	}
}

func TestEventCancellationAndIgnoreCancelled(t *testing.T) {
	bus := NewBus()
	var executed []string

	Subscribe(bus, func(e *customTestEvent) {
		executed = append(executed, "LOWEST")
	}, ListenerOptions{Priority: PriorityLowest})

	Subscribe(bus, func(e *customTestEvent) {
		executed = append(executed, "NORMAL-CANCEL")
		e.Cancel()
	}, ListenerOptions{Priority: PriorityNormal})

	// This listener should be skipped because IgnoreCancelled is true
	Subscribe(bus, func(e *customTestEvent) {
		executed = append(executed, "HIGH-IGNORED")
	}, ListenerOptions{Priority: PriorityHigh, IgnoreCancelled: true})

	// This listener should still run because IgnoreCancelled is false
	Subscribe(bus, func(e *customTestEvent) {
		executed = append(executed, "HIGHEST-STILL-RUNS")
	}, ListenerOptions{Priority: PriorityHighest, IgnoreCancelled: false})

	// Monitor should observe cancellation
	var monitorSawCancelled bool
	Subscribe(bus, func(e *customTestEvent) {
		executed = append(executed, "MONITOR")
		monitorSawCancelled = e.Cancelled()
	}, ListenerOptions{Priority: PriorityMonitor})

	ev := &customTestEvent{message: "cancel-test"}
	Publish(bus, ev)

	expected := []string{"LOWEST", "NORMAL-CANCEL", "HIGHEST-STILL-RUNS", "MONITOR"}
	if len(executed) != len(expected) {
		t.Fatalf("expected execution %v, got %v", expected, executed)
	}
	for i, name := range expected {
		if executed[i] != name {
			t.Errorf("step %d: expected %s, got %s", i, name, executed[i])
		}
	}

	if !monitorSawCancelled {
		t.Errorf("expected monitor to observe event cancellation")
	}
}

func TestEventUnsubscribe(t *testing.T) {
	bus := NewBus()
	var count int

	sub := On(bus, func(e *customTestEvent) {
		count++
	})

	Publish(bus, &customTestEvent{})
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	sub.Unsubscribe()
	Publish(bus, &customTestEvent{})
	if count != 1 {
		t.Errorf("expected count to remain 1 after unsubscribe, got %d", count)
	}
}

func TestConcurrentEventPublish(t *testing.T) {
	bus := NewBus()
	const iterations = 10000
	const goroutines = 20

	var totalCounter atomic.Int64

	// Register 5 listeners with different priorities
	for i := 0; i < 5; i++ {
		p := Priority(i % 5)
		Subscribe(bus, func(e *customTestEvent) {
			totalCounter.Add(1)
		}, ListenerOptions{Priority: p})
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations/goroutines; i++ {
				Publish(bus, &customTestEvent{message: "stress"})
			}
		}()
	}

	wg.Wait()

	// 10,000 events * 5 listeners = 50,000 executions
	expected := int64(iterations * 5)
	if totalCounter.Load() != expected {
		t.Errorf("expected %d total executions, got %d", expected, totalCounter.Load())
	}
}

type customResettableEvent struct {
	cancelled bool
}

func (e *customResettableEvent) Cancelled() bool             { return e.cancelled }
func (e *customResettableEvent) Cancel()                     { e.cancelled = true }
func (e *customResettableEvent) Uncancel()                   { e.cancelled = false }
func (e *customResettableEvent) SetCancelled(cancelled bool) { e.cancelled = cancelled }

func TestBusPanicRecoveryAndMonitorInvariance(t *testing.T) {
	bus := NewBus()
	var executed []string

	Subscribe(bus, func(e *customResettableEvent) {
		executed = append(executed, "NORMAL_PANIC")
		panic("listener exploded")
	}, ListenerOptions{Priority: PriorityNormal})

	Subscribe(bus, func(e *customResettableEvent) {
		executed = append(executed, "HIGH_RUNS")
	}, ListenerOptions{Priority: PriorityHigh})

	Subscribe(bus, func(e *customResettableEvent) {
		executed = append(executed, "MONITOR_ILLEGAL_CANCEL")
		e.Cancel() // Monitor trying to cancel
	}, ListenerOptions{Priority: PriorityMonitor})

	ev := &customResettableEvent{cancelled: false}
	Publish(bus, ev)

	if len(executed) != 3 {
		t.Fatalf("expected 3 listeners executed despite panic, got %v", executed)
	}

	// Verify invariant: Monitor cancellation attempt was reverted
	if ev.Cancelled() {
		t.Errorf("expected Monitor cancellation to be reverted, but event was cancelled")
	}
}

func TestTypedBusPriorityAndCancellation(t *testing.T) {
	tbus := NewTypedBus[customResettableEvent]()
	var executed []string

	tbus.Subscribe(func(e *customResettableEvent) {
		executed = append(executed, "LOWEST")
	}, ListenerOptions{Priority: PriorityLowest})

	tbus.Subscribe(func(e *customResettableEvent) {
		executed = append(executed, "NORMAL_CANCEL")
		e.Cancel()
	}, ListenerOptions{Priority: PriorityNormal})

	tbus.Subscribe(func(e *customResettableEvent) {
		executed = append(executed, "HIGH_IGNORED")
	}, ListenerOptions{Priority: PriorityHigh, IgnoreCancelled: true})

	tbus.Subscribe(func(e *customResettableEvent) {
		executed = append(executed, "MONITOR_ILLEGAL_UNCANCEL")
		e.Uncancel() // Monitor trying to uncancel
	}, ListenerOptions{Priority: PriorityMonitor})

	ev := &customResettableEvent{}
	tbus.Publish(ev)

	expected := []string{"LOWEST", "NORMAL_CANCEL", "MONITOR_ILLEGAL_UNCANCEL"}
	if len(executed) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, executed)
	}
	for i, name := range expected {
		if executed[i] != name {
			t.Errorf("step %d: expected %s, got %s", i, name, executed[i])
		}
	}

	// Verify monitor uncancel was reverted; event remains cancelled
	if !ev.Cancelled() {
		t.Errorf("expected event to remain cancelled despite monitor uncancel attempt")
	}
}

func TestTypedBusPanicRecovery(t *testing.T) {
	tbus := NewTypedBus[customResettableEvent]()
	var afterPanicRan bool

	tbus.Subscribe(func(e *customResettableEvent) {
		panic("typed listener panic")
	}, ListenerOptions{Priority: PriorityLowest})

	tbus.Subscribe(func(e *customResettableEvent) {
		afterPanicRan = true
	}, ListenerOptions{Priority: PriorityNormal})

	ev := &customResettableEvent{}
	tbus.Publish(ev)

	if !afterPanicRan {
		t.Errorf("expected listener after panic to still execute")
	}
}

func BenchmarkTypedBusDispatch(b *testing.B) {
	tbus := NewTypedBus[customResettableEvent]()
	tbus.Subscribe(func(e *customResettableEvent) {
		// no-op listener
	}, ListenerOptions{Priority: PriorityNormal})

	ev := &customResettableEvent{}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tbus.Publish(ev)
	}
}

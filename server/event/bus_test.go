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

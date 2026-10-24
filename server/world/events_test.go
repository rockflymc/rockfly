package world_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

func execWorld(w *world.World, f func(tx *world.Tx)) {
	_, _ = world.Call(context.Background(), w, func(tx *world.Tx) (struct{}, error) {
		f(tx)
		return struct{}{}, nil
	})
}

func TestSoundPlayEventBus(t *testing.T) {
	w := world.Config{}.New()
	bus := event.DefaultBus()

	var called atomic.Bool
	sub := event.Subscribe(bus, func(e *world.SoundPlayEvent) {
		called.Store(true)
	})
	defer sub.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		ctx := tx.Event()
		w.Handler().HandleSound(ctx, sound.Click{}, mgl64.Vec3{0, 64, 0})

		if !called.Load() {
			t.Fatal("expected SoundPlayEvent listener to be called")
		}
		if ctx.Cancelled() {
			t.Fatal("expected sound event to not be cancelled")
		}
	})
}

func TestSoundPlayEventCancellation(t *testing.T) {
	w := world.Config{}.New()
	bus := event.DefaultBus()

	sub := event.Subscribe(bus, func(e *world.SoundPlayEvent) {
		e.Cancel()
	})
	defer sub.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		ctx := tx.Event()
		w.Handler().HandleSound(ctx, sound.Click{}, mgl64.Vec3{0, 64, 0})

		if !ctx.Cancelled() {
			t.Fatal("expected sound event to be cancelled")
		}
	})
}

type testWorldCustomHandler struct {
	world.NopHandler
	soundCalls atomic.Int32
}

func (h *testWorldCustomHandler) HandleSound(ctx *world.Context, s world.Sound, pos mgl64.Vec3) {
	h.soundCalls.Add(1)
}

func TestWorldCustomHandlerCompatibility(t *testing.T) {
	w := world.Config{}.New()
	bus := event.DefaultBus()

	custom := &testWorldCustomHandler{}
	w.Handle(custom)

	var busCalled atomic.Bool
	sub := event.Subscribe(bus, func(e *world.SoundPlayEvent) {
		busCalled.Store(true)
	})
	defer sub.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		ctx := tx.Event()
		w.Handler().HandleSound(ctx, sound.Click{}, mgl64.Vec3{0, 64, 0})

		if !busCalled.Load() {
			t.Fatal("expected event bus listener to be called")
		}
		if custom.soundCalls.Load() != 1 {
			t.Fatalf("expected custom handler to be called once, got %d", custom.soundCalls.Load())
		}
	})
}

func TestLiquidFlowEventCancellation(t *testing.T) {
	w := world.Config{}.New()
	bus := event.DefaultBus()

	sub := event.Subscribe(bus, func(e *world.LiquidFlowEvent) {
		if e.Into.Y() < 10 {
			e.Cancel()
		}
	})
	defer sub.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		// Valid flow
		ctx1 := tx.Event()
		w.Handler().HandleLiquidFlow(ctx1, cube.Pos{0, 64, 0}, cube.Pos{0, 63, 0}, nil, nil)
		if ctx1.Cancelled() {
			t.Fatal("expected flow above Y=10 to be allowed")
		}

		// Blocked flow
		ctx2 := tx.Event()
		w.Handler().HandleLiquidFlow(ctx2, cube.Pos{0, 10, 0}, cube.Pos{0, 9, 0}, nil, nil)
		if !ctx2.Cancelled() {
			t.Fatal("expected flow into Y=9 to be cancelled")
		}
	})
}

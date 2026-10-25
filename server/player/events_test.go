package player_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

func execWorld(w *world.World, f func(tx *world.Tx)) {
	_, _ = world.Call(context.Background(), w, func(tx *world.Tx) (struct{}, error) {
		f(tx)
		return struct{}{}, nil
	})
}

func createTestPlayer(t *testing.T) (*world.World, *world.EntityHandle) {
	t.Helper()
	w := world.Config{}.New()
	var handle *world.EntityHandle
	execWorld(w, func(tx *world.Tx) {
		opts := world.EntitySpawnOpts{Position: mgl64.Vec3{0, 60, 0}, ID: uuid.New()}
		handle = opts.New(player.Type, player.Config{Name: "Steve"})
		tx.AddEntity(handle)
	})
	if handle == nil {
		t.Fatal("failed to instantiate test player handle")
	}
	return w, handle
}

func TestPlayerChatEventBus(t *testing.T) {
	w, h := createTestPlayer(t)
	bus := event.DefaultBus()

	var received string
	sub := event.Subscribe(bus, func(e *player.PlayerChatEvent) {
		received = e.Message
		e.Message = strings.ToUpper(e.Message)
	})
	defer sub.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		ent, ok := h.Entity(tx)
		if !ok {
			t.Fatal("failed to resolve entity")
		}
		p := ent.(*player.Player)

		msg := "hello world"
		ctx := player.NewEventContext(tx, p)
		p.Handler().HandleChat(ctx, &msg)

		if ctx.Cancelled() {
			t.Fatal("expected chat to not be cancelled")
		}
		if received != "hello world" {
			t.Fatalf("expected received message 'hello world', got %q", received)
		}
		if msg != "HELLO WORLD" {
			t.Fatalf("expected transformed message 'HELLO WORLD', got %q", msg)
		}
	})
}

func TestPlayerChatEventCancellation(t *testing.T) {
	w, h := createTestPlayer(t)
	bus := event.DefaultBus()

	sub := event.Subscribe(bus, func(e *player.PlayerChatEvent) {
		if strings.Contains(e.Message, "badword") {
			e.Cancel()
		}
	})
	defer sub.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		ent, ok := h.Entity(tx)
		if !ok {
			t.Fatal("failed to resolve entity")
		}
		p := ent.(*player.Player)

		msg := "this contains badword!"
		ctx := player.NewEventContext(tx, p)
		p.Handler().HandleChat(ctx, &msg)

		if !ctx.Cancelled() {
			t.Fatal("expected chat event to be cancelled")
		}
	})
}

func TestPlayerPriorityOrder(t *testing.T) {
	w, h := createTestPlayer(t)
	bus := event.DefaultBus()

	var order []string

	sub1 := event.Subscribe(bus, func(e *player.PlayerChatEvent) {
		order = append(order, "highest")
	}, event.ListenerOptions{Priority: event.PriorityHighest})
	defer sub1.Unsubscribe()

	sub2 := event.Subscribe(bus, func(e *player.PlayerChatEvent) {
		order = append(order, "lowest")
	}, event.ListenerOptions{Priority: event.PriorityLowest})
	defer sub2.Unsubscribe()

	sub3 := event.Subscribe(bus, func(e *player.PlayerChatEvent) {
		order = append(order, "normal")
	}, event.ListenerOptions{Priority: event.PriorityNormal})
	defer sub3.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		ent, ok := h.Entity(tx)
		if !ok {
			t.Fatal("failed to resolve entity")
		}
		p := ent.(*player.Player)

		msg := "priority test"
		ctx := player.NewEventContext(tx, p)
		p.Handler().HandleChat(ctx, &msg)

		if len(order) != 3 || order[0] != "lowest" || order[1] != "normal" || order[2] != "highest" {
			t.Fatalf("expected order [lowest, normal, highest], got %v", order)
		}
	})
}

type testCustomHandler struct {
	player.NopHandler
	chatCalls atomic.Int32
}

func (h *testCustomHandler) HandleChat(ctx *player.Context, message *string) {
	h.chatCalls.Add(1)
}

func TestPlayerCustomHandlerCompatibility(t *testing.T) {
	w, h := createTestPlayer(t)
	bus := event.DefaultBus()

	custom := &testCustomHandler{}

	var busCalled atomic.Bool
	sub := event.Subscribe(bus, func(e *player.PlayerChatEvent) {
		busCalled.Store(true)
	})
	defer sub.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		ent, ok := h.Entity(tx)
		if !ok {
			t.Fatal("failed to resolve entity")
		}
		p := ent.(*player.Player)
		p.Handle(custom)

		msg := "compat test"
		ctx := player.NewEventContext(tx, p)
		p.Handler().HandleChat(ctx, &msg)

		if !busCalled.Load() {
			t.Fatal("expected event bus listener to be called")
		}
		if custom.chatCalls.Load() != 1 {
			t.Fatalf("expected custom handler to be called once, got %d", custom.chatCalls.Load())
		}
	})
}

func TestPlayerMoveEventCancellation(t *testing.T) {
	w, h := createTestPlayer(t)
	bus := event.DefaultBus()

	sub := event.Subscribe(bus, func(e *player.PlayerMoveEvent) {
		// Cancel movement if moving past Z = 100
		if e.To.Z() > 100 {
			e.Cancel()
		}
	})
	defer sub.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		ent, ok := h.Entity(tx)
		if !ok {
			t.Fatal("failed to resolve entity")
		}
		p := ent.(*player.Player)

		// Valid move
		ctx1 := player.NewEventContext(tx, p)
		p.Handler().HandleMove(ctx1, mgl64.Vec3{0, 60, 50}, cube.Rotation{})
		if ctx1.Cancelled() {
			t.Fatal("expected move to Z=50 to be allowed")
		}

		// Blocked move
		ctx2 := player.NewEventContext(tx, p)
		p.Handler().HandleMove(ctx2, mgl64.Vec3{0, 60, 150}, cube.Rotation{})
		if !ctx2.Cancelled() {
			t.Fatal("expected move to Z=150 to be cancelled")
		}
	})
}

func TestPlayerMoveBusTypedDispatch(t *testing.T) {
	w, h := createTestPlayer(t)

	var moveCount int
	sub := player.MoveBus.Subscribe(func(e *player.PlayerMoveEvent) {
		moveCount++
		if e.To.X() > 50 {
			e.Cancel()
		}
	})
	defer sub.Unsubscribe()

	execWorld(w, func(tx *world.Tx) {
		ent, ok := h.Entity(tx)
		if !ok {
			t.Fatal("failed to resolve entity")
		}
		p := ent.(*player.Player)

		ctx := player.NewEventContext(tx, p)
		p.Handler().HandleMove(ctx, mgl64.Vec3{10, 60, 0}, cube.Rotation{})
		if ctx.Cancelled() {
			t.Fatalf("expected move to X=10 to succeed")
		}

		ctx2 := player.NewEventContext(tx, p)
		p.Handler().HandleMove(ctx2, mgl64.Vec3{60, 60, 0}, cube.Rotation{})
		if !ctx2.Cancelled() {
			t.Fatalf("expected move to X=60 to be cancelled by MoveBus")
		}

		if moveCount != 2 {
			t.Fatalf("expected MoveBus to have received 2 dispatches, got %d", moveCount)
		}
	})
}

func TestPlayerNilTxSafetyAndBridgeReuse(t *testing.T) {
	w, h := createTestPlayer(t)

	execWorld(w, func(tx *world.Tx) {
		ent, ok := h.Entity(tx)
		if !ok {
			t.Fatal("failed to resolve entity")
		}
		p := ent.(*player.Player)

		// Test bridge reuse
		h1 := p.Handler()
		h2 := p.Handler()
		if h1 != h2 {
			t.Errorf("expected p.Handler() to return cached bridge instance")
		}

		// Test nil tx safety
		ctx := player.NewEventContext(tx, p)
		p.Handler().HandleStartBreak(ctx, cube.Pos{0, 50, 0})
		p.Handler().HandleBlockBreak(ctx, cube.Pos{0, 50, 0}, nil, nil)
	})

	// Test nil Player safety
	var nilPlayer *player.Player
	nilHandler := nilPlayer.Handler()
	if nilHandler == nil {
		t.Fatalf("expected nilPlayer.Handler() to return NopHandler, got nil")
	}
	nilHandler.HandleMove(nil, mgl64.Vec3{}, cube.Rotation{})
}

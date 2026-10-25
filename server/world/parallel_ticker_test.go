package world

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

type dummyParallelEntity struct {
	NopViewer
	pos mgl64.Vec3
	rot cube.Rotation
	h   *EntityHandle
}

func (d *dummyParallelEntity) Close() error            { return nil }
func (d *dummyParallelEntity) H() *EntityHandle        { return d.h }
func (d *dummyParallelEntity) Position() mgl64.Vec3    { return d.pos }
func (d *dummyParallelEntity) Rotation() cube.Rotation { return d.rot }

func TestParallelWorldTicker_MultiWorldTicking(t *testing.T) {
	ticker := NewParallelWorldTicker(64)
	defer ticker.Close()

	cOverworld := Config{Dim: Overworld, Synchronous: true}
	cNether := Config{Dim: Nether, Synchronous: true}
	cEnd := Config{Dim: End, Synchronous: true}

	wOverworld := cOverworld.New()
	defer wOverworld.Close()
	wNether := cNether.New()
	defer wNether.Close()
	wEnd := cEnd.New()
	defer wEnd.Close()

	ticker.RegisterWorld("overworld", wOverworld)
	ticker.RegisterWorld("nether", wNether)
	ticker.RegisterWorld("end", wEnd)

	var startOverworld, startNether, startEnd int64
	wOverworld.Do(func(tx *Tx) { startOverworld = tx.CurrentTick() }).Wait(context.Background())
	wNether.Do(func(tx *Tx) { startNether = tx.CurrentTick() }).Wait(context.Background())
	wEnd.Do(func(tx *Tx) { startEnd = tx.CurrentTick() }).Wait(context.Background())

	// Tick all 3 worlds simultaneously 10 times
	for i := 0; i < 10; i++ {
		ticker.TickSynchronousParallel()
	}

	wOverworld.Do(func(tx *Tx) {
		if tx.CurrentTick() != startOverworld+10 {
			t.Fatalf("expected overworld tick %d, got %d", startOverworld+10, tx.CurrentTick())
		}
	}).Wait(context.Background())

	wNether.Do(func(tx *Tx) {
		if tx.CurrentTick() != startNether+10 {
			t.Fatalf("expected nether tick %d, got %d", startNether+10, tx.CurrentTick())
		}
	}).Wait(context.Background())

	wEnd.Do(func(tx *Tx) {
		if tx.CurrentTick() != startEnd+10 {
			t.Fatalf("expected end tick %d, got %d", startEnd+10, tx.CurrentTick())
		}
	}).Wait(context.Background())
}

func TestParallelWorldTicker_EntityHandoff(t *testing.T) {
	ticker := NewParallelWorldTicker(64)
	defer ticker.Close()

	cSource := Config{Dim: Overworld, Synchronous: true}
	cTarget := Config{Dim: Nether, Synchronous: true}

	wSource := cSource.New()
	defer wSource.Close()
	wTarget := cTarget.New()
	defer wTarget.Close()

	ticker.RegisterWorld("overworld", wSource)
	ticker.RegisterWorld("nether", wTarget)

	var entHandle *EntityHandle
	wSource.Do(func(tx *Tx) {
		entHandle = EntitySpawnOpts{Position: mgl64.Vec3{0, 64, 0}}.New(testEntityType{}, testEntityConfig{})
		ent := tx.AddEntity(entHandle)
		// Remove from source in preparation for handoff
		_ = tx.RemoveEntity(ent)
	}).Wait(context.Background())

	var transferred atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)

	success := ticker.TransferEntity(EntityHandoff{
		Handle:  entHandle,
		Source:  wSource,
		Target:  wTarget,
		Pos:     mgl64.Vec3{10, 64, 10},
		Timeout: 2 * time.Second,
		OnComplete: func(e Entity) {
			transferred.Store(true)
			wg.Done()
		},
		OnFail: func(err error) {
			t.Errorf("handoff failed: %v", err)
			wg.Done()
		},
	})

	if !success {
		t.Fatal("expected handoff submission to succeed")
	}

	wg.Wait()

	if !transferred.Load() {
		t.Fatal("expected entity to be transferred to target world")
	}

	// Verify entity now exists in wTarget
	wTarget.Do(func(tx *Tx) {
		e, ok := entHandle.Entity(tx)
		if !ok || e == nil {
			t.Fatal("expected entity to be found in target world transaction")
		}
		if e.Position() != (mgl64.Vec3{10, 64, 10}) {
			t.Fatalf("expected position (10, 64, 10), got %v", e.Position())
		}
	}).Wait(context.Background())
}

func BenchmarkParallelWorldTicker_TickConcurrent(b *testing.B) {
	ticker := NewParallelWorldTicker(64)
	defer ticker.Close()

	w1 := (Config{Dim: Overworld, Synchronous: true}).New()
	defer w1.Close()
	w2 := (Config{Dim: Nether, Synchronous: true}).New()
	defer w2.Close()
	w3 := (Config{Dim: End, Synchronous: true}).New()
	defer w3.Close()

	ticker.RegisterWorld("w1", w1)
	ticker.RegisterWorld("w2", w2)
	ticker.RegisterWorld("w3", w3)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ticker.TickSynchronousParallel()
	}
}

func TestParallelWorldTicker_NilAndNaNSafety(t *testing.T) {
	var ticker *ParallelWorldTicker
	ticker.RegisterWorld("dummy", nil)
	if _, ok := ticker.Stats("dummy"); ok {
		t.Fatal("expected false on nil ticker Stats")
	}
	ticker.TickSynchronousParallel()
	ticker.Close()
	if ticker.TransferEntity(EntityHandoff{}) {
		t.Fatal("expected false on nil ticker TransferEntity")
	}

	validTicker := NewParallelWorldTicker(16)
	defer validTicker.Close()

	var failedReason error
	// Test NaN position
	success := validTicker.TransferEntity(EntityHandoff{
		Handle: &EntityHandle{},
		Source: &World{},
		Target: &World{},
		Pos:    mgl64.Vec3{0, 64, math.NaN()},
		OnFail: func(err error) {
			failedReason = err
		},
	})
	if success {
		t.Fatal("expected false on NaN position transfer")
	}
	if failedReason == nil {
		t.Fatal("expected failedReason on NaN position transfer")
	}
}

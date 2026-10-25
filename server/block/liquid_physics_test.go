package block

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type dummyPhysicsEntity struct {
	world.NopViewer
	pos      mgl64.Vec3
	vel      mgl64.Vec3
	air      time.Duration
	sneaking bool
}

func (d *dummyPhysicsEntity) Close() error                 { return nil }
func (d *dummyPhysicsEntity) H() *world.EntityHandle       { return nil }
func (d *dummyPhysicsEntity) Position() mgl64.Vec3         { return d.pos }
func (d *dummyPhysicsEntity) Rotation() cube.Rotation      { return cube.Rotation{} }
func (d *dummyPhysicsEntity) Velocity() mgl64.Vec3         { return d.vel }
func (d *dummyPhysicsEntity) SetVelocity(v mgl64.Vec3)     { d.vel = v }
func (d *dummyPhysicsEntity) AirSupply() time.Duration     { return d.air }
func (d *dummyPhysicsEntity) SetAirSupply(a time.Duration) { d.air = a }
func (d *dummyPhysicsEntity) Sneaking() bool               { return d.sneaking }

func TestBubbleColumn_EncodeDecode(t *testing.T) {
	for _, b := range allBubbleColumns() {
		name, properties := b.EncodeBlock()
		if name != "minecraft:bubble_column" {
			t.Fatalf("expected minecraft:bubble_column, got %s", name)
		}
		dragDown, ok := properties["drag_down"].(uint8)
		if !ok || (dragDown != 0 && dragDown != 1) {
			t.Fatalf("invalid drag_down property: %v", properties["drag_down"])
		}
	}
}

func TestBubbleColumn_UpwardGeyser(t *testing.T) {
	w := newTestWorld()
	sandPos := cube.Pos{0, 60, 0}
	waterPos1 := cube.Pos{0, 61, 0}
	waterPos2 := cube.Pos{0, 62, 0}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(sandPos, SoulSand{}, nil)
		tx.SetBlock(waterPos1, Water{Depth: 8, Still: true}, nil)
		tx.SetBlock(waterPos2, Water{Depth: 8, Still: true}, nil)

		// Form bubble column above soul sand
		if !CheckCreateBubbleColumn(waterPos1, tx) {
			t.Fatal("expected bubble column creation over soul sand")
		}

		bc1, ok := tx.Block(waterPos1).(BubbleColumn)
		if !ok || bc1.DragDown {
			t.Fatalf("expected upward bubble column at pos 1, got %T %+v", tx.Block(waterPos1), bc1)
		}

		// Propagate upward into pos 2
		bc1.NeighbourUpdateTick(waterPos1, waterPos1, tx)
		bc2, ok2 := tx.Block(waterPos2).(BubbleColumn)
		if !ok2 || bc2.DragDown {
			t.Fatalf("expected upward bubble column propagated to pos 2, got %T %+v", tx.Block(waterPos2), bc2)
		}

		// Test entity physics inside upward geyser
		ent := &dummyPhysicsEntity{vel: mgl64.Vec3{0, 0, 0}, air: time.Second * 2}
		bc1.EntityInside(waterPos1, tx, ent)

		if ent.Velocity()[1] <= 0 {
			t.Fatalf("expected positive upward velocity, got %v", ent.Velocity()[1])
		}
		if ent.AirSupply() != time.Second*15 {
			t.Fatalf("expected full oxygen supply restored (15s), got %v", ent.AirSupply())
		}
	}).Wait(context.Background())
}

func TestBubbleColumn_DownwardWhirlpool(t *testing.T) {
	w := newTestWorld()
	magmaPos := cube.Pos{0, 60, 0}
	waterPos := cube.Pos{0, 61, 0}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(magmaPos, Magma{}, nil)
		tx.SetBlock(waterPos, Water{Depth: 8, Still: true}, nil)

		if !CheckCreateBubbleColumn(waterPos, tx) {
			t.Fatal("expected downward whirlpool creation over magma")
		}

		bc, ok := tx.Block(waterPos).(BubbleColumn)
		if !ok || !bc.DragDown {
			t.Fatalf("expected downward bubble column, got %T %+v", tx.Block(waterPos), bc)
		}

		// Test entity drag downward
		ent := &dummyPhysicsEntity{vel: mgl64.Vec3{0, 0, 0}}
		bc.EntityInside(waterPos, tx, ent)

		if ent.Velocity()[1] >= 0 {
			t.Fatalf("expected negative downward velocity, got %v", ent.Velocity()[1])
		}
	}).Wait(context.Background())
}

func TestBubbleColumn_Dissolution(t *testing.T) {
	w := newTestWorld()
	sandPos := cube.Pos{0, 60, 0}
	waterPos := cube.Pos{0, 61, 0}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(sandPos, SoulSand{}, nil)
		tx.SetBlock(waterPos, Water{Depth: 8, Still: true}, nil)
		CheckCreateBubbleColumn(waterPos, tx)

		// Replace soul sand with stone
		tx.SetBlock(sandPos, Stone{}, nil)

		// Trigger neighbour update on bubble column
		bc, _ := tx.Block(waterPos).(BubbleColumn)
		bc.NeighbourUpdateTick(waterPos, sandPos, tx)

		// Bubble column should have dissolved back to still water
		if w, ok := tx.Block(waterPos).(Water); !ok || !w.Still || w.Depth != 8 {
			t.Fatalf("expected bubble column to dissolve into still water, got %T %+v", tx.Block(waterPos), tx.Block(waterPos))
		}
	}).Wait(context.Background())
}

func TestWaterlogging_LiquidDisplacer(t *testing.T) {
	w := newTestWorld()
	chestPos := cube.Pos{0, 64, 0}
	chest := NewChest()

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(chestPos, chest, nil)

		// Waterlog the chest on layer 1
		tx.SetLiquid(chestPos, Water{Depth: 8, Still: true})

		// Block layer 0 should remain Chest
		b := tx.Block(chestPos)
		if _, ok := b.(Chest); !ok {
			t.Fatalf("expected block on layer 0 to be Chest, got %T", b)
		}

		// Liquid layer 1 should contain water
		liq, ok := tx.Liquid(chestPos)
		if !ok {
			t.Fatal("expected liquid on layer 1 for waterlogged chest")
		}
		if w, ok := liq.(Water); !ok || !w.Still || w.Depth != 8 {
			t.Fatalf("expected water source on layer 1, got %T %+v", liq, liq)
		}
	}).Wait(context.Background())
}

func BenchmarkBubbleColumn_EntityInside(b *testing.B) {
	bc := BubbleColumn{DragDown: false}
	ent := &dummyPhysicsEntity{vel: mgl64.Vec3{0, 0, 0}}
	pos := cube.Pos{0, 64, 0}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bc.EntityInside(pos, nil, ent)
	}
}

func TestBubbleColumn_NilAndNaNSafety(t *testing.T) {
	bc := BubbleColumn{DragDown: false}
	pos := cube.Pos{0, 64, 0}

	// 1. Nil entity inside
	bc.EntityInside(pos, nil, nil)

	// 2. NaN/Inf velocity inside
	ent := &dummyPhysicsEntity{vel: mgl64.Vec3{math.NaN(), math.Inf(1), math.NaN()}}
	bc.EntityInside(pos, nil, ent)
	if math.IsNaN(ent.vel[0]) || math.IsNaN(ent.vel[1]) || math.IsNaN(ent.vel[2]) {
		t.Fatalf("expected NaN velocity to be sanitized, got %+v", ent.vel)
	}

	// 3. Nil tx safety
	bc.NeighbourUpdateTick(pos, pos, nil)
	bc.ScheduledTick(pos, nil, nil)
	if CheckCreateBubbleColumn(pos, nil) {
		t.Fatal("expected CheckCreateBubbleColumn with nil tx to return false")
	}

	// 4. World boundary out-of-bounds safety
	w := newTestWorld()
	w.Do(func(tx *world.Tx) {
		r := tx.Range()
		outPos := cube.Pos{0, r.Max() + 10, 0}
		bc.NeighbourUpdateTick(outPos, outPos, tx)
		bc.ScheduledTick(outPos, tx, nil)
		if CheckCreateBubbleColumn(outPos, tx) {
			t.Fatal("expected CheckCreateBubbleColumn with out-of-bounds pos to return false")
		}
	}).Wait(context.Background())
}

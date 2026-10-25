package world

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

func TestSpatialGrid_InsertQueryRemove(t *testing.T) {
	grid := NewSpatialGrid()

	e1 := EntitySpawnOpts{ID: uuid.New(), Position: mgl64.Vec3{10, 64, 10}}.New(dummyEntityType{name: "minecraft:zombie"}, testEntityConfig{})
	e2 := EntitySpawnOpts{ID: uuid.New(), Position: mgl64.Vec3{15, 64, 15}}.New(dummyEntityType{name: "minecraft:skeleton"}, testEntityConfig{})
	e3 := EntitySpawnOpts{ID: uuid.New(), Position: mgl64.Vec3{100, 64, 100}}.New(dummyEntityType{name: "minecraft:creeper"}, testEntityConfig{})

	grid.Insert(e1, e1.data.Pos)
	grid.Insert(e2, e2.data.Pos)
	grid.Insert(e3, e3.data.Pos)

	if count := grid.EntityCount(); count != 3 {
		t.Fatalf("expected 3 entities, got %d", count)
	}

	// Query box around (0..30, 60..70, 0..30)
	queryBox := cube.Box(0, 60, 0, 30, 70, 30)
	var found []*EntityHandle
	grid.Query(queryBox, func(h *EntityHandle) bool {
		found = append(found, h)
		return true
	})

	if len(found) != 2 {
		t.Fatalf("expected 2 entities in query box, got %d", len(found))
	}

	// Remove e1
	grid.Remove(e1)
	if count := grid.EntityCount(); count != 2 {
		t.Fatalf("expected 2 entities after remove, got %d", count)
	}

	found = found[:0]
	grid.Query(queryBox, func(h *EntityHandle) bool {
		found = append(found, h)
		return true
	})
	if len(found) != 1 || found[0] != e2 {
		t.Fatalf("expected only e2 after removing e1, got %d results", len(found))
	}
}

func TestSpatialGrid_MoveAcrossCells(t *testing.T) {
	grid := NewSpatialGrid()

	// Initial position in cell (0, 0)
	e := EntitySpawnOpts{ID: uuid.New(), Position: mgl64.Vec3{10, 64, 10}}.New(dummyEntityType{name: "minecraft:cow"}, testEntityConfig{})
	grid.Insert(e, e.data.Pos)

	c0 := CellCoordFromXZ(10, 10)
	if entities := grid.CellEntities(c0); len(entities) != 1 {
		t.Fatalf("expected 1 entity in cell (0, 0), got %d", len(entities))
	}

	// Small move within same cell (12, 64, 12)
	e.data.Pos = mgl64.Vec3{12, 64, 12}
	moved := grid.Move(e, e.data.Pos)
	if moved {
		t.Fatalf("expected Move within same cell to return false")
	}

	// Large move into cell (1, 1) -> (40, 64, 40)
	e.data.Pos = mgl64.Vec3{40, 64, 40}
	moved = grid.Move(e, e.data.Pos)
	if !moved {
		t.Fatalf("expected Move across cell boundary to return true")
	}

	if entities := grid.CellEntities(c0); len(entities) != 0 {
		t.Fatalf("expected 0 entities in old cell (0, 0), got %d", len(entities))
	}

	c1 := CellCoordFromXZ(40, 40)
	if entities := grid.CellEntities(c1); len(entities) != 1 {
		t.Fatalf("expected 1 entity in new cell (1, 1), got %d", len(entities))
	}
}

func TestSpatialGrid_QueryRadius(t *testing.T) {
	grid := NewSpatialGrid()

	center := mgl64.Vec3{50, 64, 50}
	eClose := EntitySpawnOpts{ID: uuid.New(), Position: mgl64.Vec3{55, 64, 50}}.New(dummyEntityType{name: "minecraft:sheep"}, testEntityConfig{})
	eFar := EntitySpawnOpts{ID: uuid.New(), Position: mgl64.Vec3{90, 64, 50}}.New(dummyEntityType{name: "minecraft:sheep"}, testEntityConfig{})

	grid.Insert(eClose, eClose.data.Pos)
	grid.Insert(eFar, eFar.data.Pos)

	var found []*EntityHandle
	grid.QueryRadius(center, 10.0, func(h *EntityHandle) bool {
		found = append(found, h)
		return true
	})

	if len(found) != 1 || found[0] != eClose {
		t.Fatalf("expected only eClose within radius 10, got %d results", len(found))
	}
}

func TestSpatialGrid_WorldEntitiesWithin(t *testing.T) {
	w := Config{Synchronous: true}.New()
	defer w.Close()

	var ent1, ent2 Entity
	w.Do(func(tx *Tx) {
		h1 := EntitySpawnOpts{Position: mgl64.Vec3{5, 64, 5}}.New(testEntityType{}, testEntityConfig{})
		h2 := EntitySpawnOpts{Position: mgl64.Vec3{200, 64, 200}}.New(testEntityType{}, testEntityConfig{})

		ent1 = tx.AddEntity(h1)
		ent2 = tx.AddEntity(h2)

		searchBox := cube.Box(0, 60, 0, 10, 70, 10)
		var matches []Entity
		for e := range tx.EntitiesWithin(searchBox) {
			matches = append(matches, e)
		}

		if len(matches) != 1 {
			t.Fatalf("expected 1 entity within box, got %d", len(matches))
		}
		if matches[0].H() != ent1.H() {
			t.Fatalf("expected ent1 to match")
		}

		// Remove ent1
		tx.RemoveEntity(ent1)

		matches = matches[:0]
		for e := range tx.EntitiesWithin(searchBox) {
			matches = append(matches, e)
		}
		if len(matches) != 0 {
			t.Fatalf("expected 0 entities after removal, got %d", len(matches))
		}
	})
	_ = ent2
}

func TestSpatialGrid_NegativeCoordinatesAndHardening(t *testing.T) {
	// Negative coordinates floor check
	coord := CellCoordFromVec3(mgl64.Vec3{-0.5, 64, -0.5})
	if coord.X != -1 || coord.Z != -1 {
		t.Fatalf("expected CellCoord{-1, -1} for (-0.5, -0.5), got %+v", coord)
	}

	// NaN and Inf sanitization
	coordNaN := CellCoordFromVec3(mgl64.Vec3{math.NaN(), 64, math.Inf(1)})
	if coordNaN.X != 0 || coordNaN.Z != 0 {
		t.Fatalf("expected CellCoord{0, 0} for (NaN, Inf), got %+v", coordNaN)
	}

	// Nil receiver safety
	var nilGrid *SpatialGrid
	nilGrid.Insert(nil, mgl64.Vec3{0, 0, 0})
	nilGrid.Remove(nil)
	nilGrid.Move(nil, mgl64.Vec3{0, 0, 0})
	nilGrid.Query(cube.Box(0, 0, 0, 1, 1, 1), func(h *EntityHandle) bool { return true })
	nilGrid.QueryRadius(mgl64.Vec3{0, 0, 0}, 10, func(h *EntityHandle) bool { return true })
	if nilGrid.CellEntities(CellCoord{0, 0}) != nil || nilGrid.EntityCount() != 0 || nilGrid.CellCount() != 0 {
		t.Fatalf("nil grid should return 0/nil")
	}
	nilGrid.Clear()

	// Zero-value struct safety without NewSpatialGrid
	zeroGrid := &SpatialGrid{}
	e := EntitySpawnOpts{ID: uuid.New(), Position: mgl64.Vec3{5, 64, 5}}.New(dummyEntityType{name: "minecraft:sheep"}, testEntityConfig{})
	zeroGrid.Insert(e, e.data.Pos)
	if zeroGrid.EntityCount() != 1 {
		t.Fatalf("expected 1 entity in zero-value grid, got %d", zeroGrid.EntityCount())
	}
	zeroGrid.Move(e, mgl64.Vec3{100, 64, 100})
	zeroGrid.Remove(e)
	if zeroGrid.EntityCount() != 0 {
		t.Fatalf("expected 0 entities after removal from zero-value grid")
	}

	// QueryRadius with NaN / Inf / Negative radius
	grid := NewSpatialGrid()
	grid.Insert(e, e.data.Pos)
	grid.QueryRadius(mgl64.Vec3{math.NaN(), 0, 0}, 10, func(h *EntityHandle) bool {
		t.Fatalf("should not invoke callback on NaN center")
		return true
	})
	grid.QueryRadius(mgl64.Vec3{0, 0, 0}, -5, func(h *EntityHandle) bool {
		t.Fatalf("should not invoke callback on negative radius")
		return true
	})
	grid.QueryRadius(mgl64.Vec3{0, 0, 0}, math.NaN(), func(h *EntityHandle) bool {
		t.Fatalf("should not invoke callback on NaN radius")
		return true
	})
}

func BenchmarkSpatialGrid_Query_5000Entities(b *testing.B) {
	grid := NewSpatialGrid()

	// Distribute 5,000 entities across a 500x500 area
	for i := 0; i < 5000; i++ {
		x := float64((i * 17) % 500)
		z := float64((i * 31) % 500)
		h := EntitySpawnOpts{
			ID:       uuid.New(),
			Position: mgl64.Vec3{x, 64, z},
		}.New(dummyEntityType{name: "minecraft:cow"}, testEntityConfig{})
		grid.Insert(h, h.data.Pos)
	}

	// Query a typical 16x16 bounding box (e.g. hopper suction or attack box)
	queryBox := cube.Box(250, 60, 250, 266, 70, 266)

	b.ResetTimer()
	b.ReportAllocs()

	count := 0
	for i := 0; i < b.N; i++ {
		count = 0
		grid.Query(queryBox, func(h *EntityHandle) bool {
			count++
			return true
		})
	}
}

func BenchmarkSpatialGrid_Move(b *testing.B) {
	grid := NewSpatialGrid()
	h := EntitySpawnOpts{
		ID:       uuid.New(),
		Position: mgl64.Vec3{10, 64, 10},
	}.New(dummyEntityType{name: "minecraft:pig"}, testEntityConfig{})
	grid.Insert(h, h.data.Pos)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Alternate between cell 0 and cell 1
		offset := float64((i % 2) * 40)
		grid.Move(h, mgl64.Vec3{10 + offset, 64, 10 + offset})
	}
}

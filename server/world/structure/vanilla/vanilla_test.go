package vanilla

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

func TestStructureDimensionsAndPalette(t *testing.T) {
	structures := map[Type]world.Structure{
		TypeVillage:         NewVillage(VillagePlains),
		TypePillagerOutpost: NewPillagerOutpost(),
		TypeDesertPyramid:   NewDesertPyramid(),
		TypeWitchHut:        NewWitchHut(),
		TypeMineshaft:       NewMineshaft(),
		TypeOceanMonument:   NewOceanMonument(),
		TypeStronghold:      NewStronghold(),
		TypeAncientCity:     NewAncientCity(),
		TypeNetherFortress:  NewNetherFortress(),
		TypeBastionRemnant:  NewBastionRemnant(),
	}

	for sType, s := range structures {
		dims := s.Dimensions()
		if dims[0] <= 0 || dims[1] <= 0 || dims[2] <= 0 {
			t.Fatalf("structure %s has invalid dimensions: %v", sType, dims)
		}

		// Ensure At() returns valid blocks
		var blockCount int
		for x := 0; x < dims[0]; x++ {
			for y := 0; y < dims[1]; y++ {
				for z := 0; z < dims[2]; z++ {
					b, _ := s.At(x, y, z, nil)
					if b != nil {
						blockCount++
					}
				}
			}
		}
		if blockCount == 0 {
			t.Fatalf("structure %s has zero populated blocks", sType)
		}
	}
}

func TestStructureBoundingBoxesAndManager(t *testing.T) {
	mgr := NewManager()
	origin := cube.Pos{100, 64, 100}

	inst := &Instance{
		Type:   TypeDesertPyramid,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(cube.Pos{21, 19, 21})),
		Pieces: []Piece{
			{Name: "secret_chamber", BBox: NewBoundingBox(origin.Add(cube.Pos{8, 0, 8}), origin.Add(cube.Pos{12, 6, 12}))},
		},
	}
	mgr.Add(inst)

	// Test InsideStructure
	insidePos := cube.Pos{105, 70, 105}
	if !mgr.InsideStructure(insidePos, TypeDesertPyramid) {
		t.Fatalf("expected pos %v to be inside DesertPyramid", insidePos)
	}

	outsidePos := cube.Pos{50, 70, 50}
	if mgr.InsideStructure(outsidePos, TypeDesertPyramid) {
		t.Fatalf("expected pos %v to be outside DesertPyramid", outsidePos)
	}

	// Test InsidePiece
	chamberPos := cube.Pos{110, 65, 110}
	if !mgr.InsidePiece(chamberPos, TypeDesertPyramid, "secret_chamber") {
		t.Fatalf("expected pos %v to be inside secret_chamber piece", chamberPos)
	}
}

func TestStructureGridPlacement(t *testing.T) {
	seed := int64(123456789)

	// Check determinism across multiple calls
	c1 := world.ChunkPos{10, 10}
	res1 := IsStructureChunk(seed, c1, TypeVillage)
	res2 := IsStructureChunk(seed, c1, TypeVillage)
	if res1 != res2 {
		t.Fatalf("expected deterministic result for IsStructureChunk, got %v and %v", res1, res2)
	}

	// Verify that within a 34x34 cell there is at most one village candidate chunk
	cfg := DefaultPlacements[TypeVillage]
	candChunkX, candChunkZ := StructureChunkAtCell(seed, 0, 0, cfg)
	if candChunkX < 0 || candChunkX >= cfg.Spacing || candChunkZ < 0 || candChunkZ >= cfg.Spacing {
		t.Fatalf("candidate chunk coordinates out of range: (%d, %d)", candChunkX, candChunkZ)
	}

	if !IsStructureChunk(seed, world.ChunkPos{int32(candChunkX), int32(candChunkZ)}, TypeVillage) {
		t.Fatalf("expected chunk (%d, %d) to be a village chunk", candChunkX, candChunkZ)
	}
}

func TestStructureLocate(t *testing.T) {
	seed := int64(987654321)
	origin := cube.Pos{0, 64, 0}

	pos, found := Locate(seed, origin, TypeVillage, 100, nil)
	if !found {
		t.Fatalf("expected to locate a village within 100 chunks")
	}

	chunkX := pos.X() >> 4
	chunkZ := pos.Z() >> 4
	if !IsStructureChunk(seed, world.ChunkPos{int32(chunkX), int32(chunkZ)}, TypeVillage) {
		t.Fatalf("located pos %v does not correspond to a valid structure chunk", pos)
	}
}

func TestStructureBuildChunk(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry
	c := chunk.New(reg, cube.Range{-64, 319})

	pyramid := NewDesertPyramid()
	chunkPos := world.ChunkPos{0, 0}
	origin := cube.Pos{0, 64, 0}

	pyramid.BuildChunk(c, chunkPos, origin, reg)

	// Verify that blocks were set in chunk
	var nonAirCount int
	airID := reg.BlockRuntimeID(block.Air{})
	for x := range 16 {
		for z := range 16 {
			for y := int16(64); y < 80; y++ {
				if c.Block(uint8(x), y, uint8(z), 0) != airID {
					nonAirCount++
				}
			}
		}
	}

	if nonAirCount == 0 {
		t.Fatalf("expected non-air blocks to be placed into chunk by BuildChunk")
	}
}

func BenchmarkStructureBuildChunk(b *testing.B) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry
	c := chunk.New(reg, cube.Range{-64, 319})
	pyramid := NewDesertPyramid()
	chunkPos := world.ChunkPos{0, 0}
	origin := cube.Pos{0, 64, 0}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pyramid.BuildChunk(c, chunkPos, origin, reg)
	}
}

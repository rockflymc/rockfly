package generator_test

import (
	"sync"
	"testing"

	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator"
)

func TestProtoChunkStagesAndStructures(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	c := chunk.New(world.DefaultBlockRegistry, cube.Range{-64, 319})
	pos := chunk.Pos{2, -3}
	proto := generator.NewProtoChunk(pos, c)

	if proto.Pos() != pos {
		t.Fatalf("expected pos %v, got %v", pos, proto.Pos())
	}
	if proto.Stage() != generator.StageEmpty {
		t.Fatalf("expected initial stage Empty, got %v", proto.Stage())
	}

	proto.SetStage(generator.StageBiomes)
	if proto.Stage() != generator.StageBiomes {
		t.Fatalf("expected stage Biomes, got %v", proto.Stage())
	}

	proto.SetStructureStart("village", "plains_center")
	data, ok := proto.StructureStart("village")
	if !ok || data != "plains_center" {
		t.Fatalf("expected structure start plains_center, got %v (ok=%v)", data, ok)
	}

	if _, ok := proto.StructureStart("non_existent"); ok {
		t.Fatalf("expected non_existent structure to return false")
	}
}

func TestWorldGenRegionBoundsAndCrossChunkEditing(t *testing.T) {
	chunks := make(map[chunk.Pos]*generator.ProtoChunk)
	center := chunk.Pos{10, 20}
	radius := 1 // 3x3 region: X in [9..11], Z in [19..21]

	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			p := chunk.Pos{center[0] + int32(dx), center[1] + int32(dz)}
			c := chunk.New(world.DefaultBlockRegistry, cube.Range{-64, 319})
			chunks[p] = generator.NewProtoChunk(p, c)
		}
	}

	reg, err := generator.NewWorldGenRegion(center, radius, chunks, generator.StageFeatures)
	if err != nil {
		t.Fatalf("failed to create WorldGenRegion: %v", err)
	}

	if reg.Center() != center || reg.Radius() != radius {
		t.Fatalf("region mismatch: center=%v, radius=%d", reg.Center(), reg.Radius())
	}

	// Test InRange
	// Center chunk (10, 20) world coords: [160..175], [320..335]
	if !reg.InRange(160, 320) {
		t.Fatalf("expected center coords (160, 320) to be in range")
	}
	// Neighbor chunk (9, 20) world coords: [144..159]
	if !reg.InRange(150, 325) {
		t.Fatalf("expected neighbor coords (150, 325) to be in range")
	}
	// Out-of-bounds chunk (8, 20) world coords: [128..143]
	if reg.InRange(140, 320) {
		t.Fatalf("expected out-of-bounds coords (140, 320) to be out of range")
	}

	// Test cross-chunk block placement:
	// Place a block at (159, 64, 320) which is in chunk (9, 20)
	// Place another at (160, 64, 320) which is in chunk (10, 20)
	testRuntimeID := uint32(42)
	if err := reg.SetBlock(159, 64, 320, testRuntimeID); err != nil {
		t.Fatalf("failed to set block across chunk border: %v", err)
	}
	if err := reg.SetBlock(160, 64, 320, testRuntimeID+1); err != nil {
		t.Fatalf("failed to set block in center chunk: %v", err)
	}

	val1, err := reg.Block(159, 64, 320)
	if err != nil || val1 != testRuntimeID {
		t.Fatalf("expected runtime ID %d, got %d (err: %v)", testRuntimeID, val1, err)
	}
	val2, err := reg.Block(160, 64, 320)
	if err != nil || val2 != testRuntimeID+1 {
		t.Fatalf("expected runtime ID %d, got %d (err: %v)", testRuntimeID+1, val2, err)
	}

	// Test out of range error
	if err := reg.SetBlock(500, 64, 500, 1); err == nil {
		t.Fatalf("expected error setting block out of region bounds")
	}
	if _, err := reg.Block(500, 64, 500); err == nil {
		t.Fatalf("expected error reading block out of region bounds")
	}
}

func TestWorldGenRegionMissingChunkValidation(t *testing.T) {
	chunks := make(map[chunk.Pos]*generator.ProtoChunk)
	center := chunk.Pos{0, 0}
	// Missing chunks in 3x3
	_, err := generator.NewWorldGenRegion(center, 1, chunks, generator.StageFeatures)
	if err == nil {
		t.Fatalf("expected error when chunks are missing from region")
	}
}

func TestProtoChunkConcurrentAccess(t *testing.T) {
	c := chunk.New(world.DefaultBlockRegistry, cube.Range{-64, 319})
	proto := generator.NewProtoChunk(chunk.Pos{0, 0}, c)

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			proto.SetStage(generator.StageFeatures)
			proto.SetStructureStart("feature", id)
			_ = proto.Stage()
			_, _ = proto.StructureStart("feature")
		}(i)
	}
	wg.Wait()
}

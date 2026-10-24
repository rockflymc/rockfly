package end_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/end"
)

func TestEndDimensions(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry
	endStoneRID := reg.BlockRuntimeID(block.EndStone{})
	airRID := reg.BlockRuntimeID(block.Air{})

	gen := end.NewEnd(12345)

	// 1. Central Island (ChunkPos {0, 0})
	cCenter := chunk.New(reg, cube.Range{0, 255})
	gen.GenerateChunk(world.ChunkPos{0, 0}, cCenter)

	centerBlock := cCenter.Block(0, 60, 0, 0)
	if centerBlock != endStoneRID {
		t.Fatalf("expected central island core to be end stone, got %d", centerBlock)
	}

	// 2. Void Gap (ChunkPos {30, 0} -> X around 480 blocks, in the 120..1000 void ring)
	cVoid := chunk.New(reg, cube.Range{0, 255})
	gen.GenerateChunk(world.ChunkPos{30, 0}, cVoid)

	for cx := range 16 {
		for cz := range 16 {
			for y := 0; y <= 128; y++ {
				if b := cVoid.Block(uint8(cx), int16(y), uint8(cz), 0); b != airRID {
					t.Fatalf("expected void gap to be pure air, found block %d at (%d, %d, %d)", b, cx, y, cz)
				}
			}
		}
	}

	// 3. Outer Islands (ChunkPos {100, 0} -> X = 1600 blocks)
	cOuter := chunk.New(reg, cube.Range{0, 255})
	gen.GenerateChunk(world.ChunkPos{100, 0}, cOuter)

	var outerStoneCount int
	for cx := range 16 {
		for cz := range 16 {
			for y := 45; y <= 75; y++ {
				if cOuter.Block(uint8(cx), int16(y), uint8(cz), 0) == endStoneRID {
					outerStoneCount++
				}
			}
		}
	}
	if outerStoneCount == 0 {
		t.Fatal("expected outer islands to generate at X=1600")
	}
}

func BenchmarkEndGenerateChunk(b *testing.B) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	gen := end.NewEnd(12345)
	c := chunk.New(reg, cube.Range{0, 255})
	pos := world.ChunkPos{0, 0}

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		pos[0] = int32(i)
		gen.GenerateChunk(pos, c)
	}
}

package nether_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/nether"
)

func TestNetherBoundaries(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	var nilNether *nether.Nether
	nilNether.GenerateChunk(world.ChunkPos{0, 0}, nil)

	bedrockRID := reg.BlockRuntimeID(block.Bedrock{})
	lavaRID := reg.BlockRuntimeID(block.Lava{Still: true, Depth: 8})

	gen := nether.NewNether(12345)
	c := chunk.New(reg, cube.Range{0, 127})

	gen.GenerateChunk(world.ChunkPos{0, 0}, c)

	for cx := range 16 {
		for cz := range 16 {
			// Floor at Y=0 must be solid bedrock
			if b := c.Block(uint8(cx), 0, uint8(cz), 0); b != bedrockRID {
				t.Fatalf("expected bedrock at floor (Y=0), got %d", b)
			}
			// Ceiling at Y=127 must be solid bedrock
			if b := c.Block(uint8(cx), 127, uint8(cz), 0); b != bedrockRID {
				t.Fatalf("expected bedrock at ceiling (Y=127), got %d", b)
			}
		}
	}

	// Verify lava ocean is present in low open cavities
	var lavaFound bool
	for cx := range 16 {
		for cz := range 16 {
			for y := 1; y <= 31; y++ {
				if c.Block(uint8(cx), int16(y), uint8(cz), 0) == lavaRID {
					lavaFound = true
					break
				}
			}
		}
	}
	if !lavaFound {
		t.Fatal("expected lava ocean to be generated below Y=31")
	}
}

func BenchmarkNetherGenerateChunk(b *testing.B) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	gen := nether.NewNether(12345)
	c := chunk.New(reg, cube.Range{0, 127})
	pos := world.ChunkPos{0, 0}

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		pos[0] = int32(i)
		gen.GenerateChunk(pos, c)
	}
}

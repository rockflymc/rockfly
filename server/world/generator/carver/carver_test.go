package carver_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/carver"
)

func TestCarverBoundaries(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	c := carver.NewNoiseCarver(12345)

	// High altitude must never be carved
	if c.ShouldCarve(0, 100, 0) {
		t.Fatal("expected no caves at Y=100")
	}

	// Below bedrock floor must never be carved
	if c.ShouldCarve(0, -64, 0) {
		t.Fatal("expected no caves at Y=-64")
	}
}

func TestAquiferLavaSea(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	air := reg.BlockRuntimeID(block.Air{})
	water := reg.BlockRuntimeID(block.Water{Still: true, Depth: 8})
	lava := reg.BlockRuntimeID(block.Lava{Still: true, Depth: 8})

	aquifer := carver.NewAquiferSampler(12345, air, water, lava)

	// Below Y = -54 must always return lava
	for y := -63; y <= -54; y++ {
		fluid := aquifer.FluidAt(10, y, 10)
		if fluid != lava {
			t.Fatalf("expected lava at Y=%d, got %d (lava=%d)", y, fluid, lava)
		}
	}
}

func TestCarveChunk(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	bedrockRID := reg.BlockRuntimeID(block.Bedrock{})
	deepslateRID := reg.BlockRuntimeID(block.Deepslate{})
	stoneRID := reg.BlockRuntimeID(block.Stone{})

	ch := chunk.New(reg, cube.Range{-64, 319})

	// Fill chunk with solid terrain
	for cx := range 16 {
		for cz := range 16 {
			ch.SetBlock(uint8(cx), -64, uint8(cz), 0, bedrockRID)
			for y := -63; y < 0; y++ {
				ch.SetBlock(uint8(cx), int16(y), uint8(cz), 0, deepslateRID)
			}
			for y := 0; y <= 64; y++ {
				ch.SetBlock(uint8(cx), int16(y), uint8(cz), 0, stoneRID)
			}
		}
	}

	c := carver.NewNoiseCarver(99999)
	c.CarveChunk(0, 0, ch)

	// Bedrock at Y=-64 must remain completely intact across all 256 columns
	for cx := range 16 {
		for cz := range 16 {
			b := ch.Block(uint8(cx), -64, uint8(cz), 0)
			if b != bedrockRID {
				t.Fatalf("bedrock at (%d, -64, %d) was modified: %d", cx, cz, b)
			}
		}
	}

	// Verify subterranean cave porosity (some stone/deepslate should have been carved)
	var carvedCount int
	for cx := range 16 {
		for cz := range 16 {
			for y := -63; y <= 62; y++ {
				b := ch.Block(uint8(cx), int16(y), uint8(cz), 0)
				if b != stoneRID && b != deepslateRID {
					carvedCount++
				}
			}
		}
	}

	if carvedCount == 0 {
		t.Fatal("expected at least some subterranean blocks to be carved into caves")
	}
}

func BenchmarkCarveChunk(b *testing.B) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	bedrockRID := reg.BlockRuntimeID(block.Bedrock{})
	deepslateRID := reg.BlockRuntimeID(block.Deepslate{})
	stoneRID := reg.BlockRuntimeID(block.Stone{})

	ch := chunk.New(reg, cube.Range{-64, 319})
	for cx := range 16 {
		for cz := range 16 {
			ch.SetBlock(uint8(cx), -64, uint8(cz), 0, bedrockRID)
			for y := -63; y < 0; y++ {
				ch.SetBlock(uint8(cx), int16(y), uint8(cz), 0, deepslateRID)
			}
			for y := 0; y <= 64; y++ {
				ch.SetBlock(uint8(cx), int16(y), uint8(cz), 0, stoneRID)
			}
		}
	}

	c := carver.NewNoiseCarver(12345)

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		c.CarveChunk(i, i, ch)
	}
}

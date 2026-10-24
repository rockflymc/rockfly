package surface_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/surface"
)

func TestStratifyBedrock(t *testing.T) {
	// Y = -64 must always be bedrock
	for x := -50; x <= 50; x += 10 {
		for z := -50; z <= 50; z += 10 {
			if !surface.StratifyBedrock(x, -64, z, 12345) {
				t.Fatalf("expected bedrock at (%d, -64, %d)", x, z)
			}
		}
	}

	// Y = -59 must never be bedrock
	for x := -50; x <= 50; x += 10 {
		for z := -50; z <= 50; z += 10 {
			if surface.StratifyBedrock(x, -59, z, 12345) {
				t.Fatalf("unexpected bedrock at (%d, -59, %d)", x, z)
			}
		}
	}

	// Y = -62 should have both bedrock and non-bedrock (dither)
	var bedrockCount, nonBedrockCount int
	for x := range 100 {
		for z := range 100 {
			if surface.StratifyBedrock(x, -62, z, 999) {
				bedrockCount++
			} else {
				nonBedrockCount++
			}
		}
	}

	if bedrockCount == 0 || nonBedrockCount == 0 {
		t.Fatalf("expected dither at Y=-62, got bedrock=%d, nonBedrock=%d", bedrockCount, nonBedrockCount)
	}
}

func TestDeepslateDither(t *testing.T) {
	if !surface.DeepslateDither(0, -1, 0, 12345) {
		t.Fatal("expected deepslate below Y=0")
	}
	if surface.DeepslateDither(0, 9, 0, 12345) {
		t.Fatal("expected no deepslate above Y=8")
	}

	var deepslateCount, stoneCount int
	for x := range 100 {
		for z := range 100 {
			if surface.DeepslateDither(x, 4, z, 777) {
				deepslateCount++
			} else {
				stoneCount++
			}
		}
	}

	if deepslateCount == 0 || stoneCount == 0 {
		t.Fatalf("expected dither at Y=4, got deepslate=%d, stone=%d", deepslateCount, stoneCount)
	}
}

func TestBuildSurface(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	grassRID := reg.BlockRuntimeID(block.Grass{})
	dirtRID := reg.BlockRuntimeID(block.Dirt{})
	sandRID := reg.BlockRuntimeID(block.Sand{Red: false})
	sandstoneRID := reg.BlockRuntimeID(block.Sandstone{})
	stoneRID := reg.BlockRuntimeID(block.Stone{})
	bedrockRID := reg.BlockRuntimeID(block.Bedrock{})

	builder := surface.NewRuleBuilder(12345)

	// Test 1: Plains column
	ch := chunk.New(reg, cube.Range{-64, 319})
	ch.SetBiome(0, 64, 0, uint32(biome.Plains{}.EncodeBiome()))
	for y := -64; y <= 70; y++ {
		ch.SetBlock(0, int16(y), 0, 0, stoneRID)
	}

	builder.BuildSurface(0, 0, ch)

	// Top block at Y=70 should be Grass
	topBlock := ch.Block(0, 70, 0, 0)
	if topBlock != grassRID {
		t.Fatalf("expected grass at Y=70, got %d", topBlock)
	}
	// Underneath should be Dirt
	underBlock := ch.Block(0, 69, 0, 0)
	if underBlock != dirtRID {
		t.Fatalf("expected dirt at Y=69, got %d", underBlock)
	}
	// Floor at Y=-64 must be Bedrock
	floorBlock := ch.Block(0, -64, 0, 0)
	if floorBlock != bedrockRID {
		t.Fatalf("expected bedrock at Y=-64, got %d", floorBlock)
	}

	// Test 2: Desert column
	chDesert := chunk.New(reg, cube.Range{-64, 319})
	chDesert.SetBiome(0, 64, 0, uint32(biome.Desert{}.EncodeBiome()))
	for y := -64; y <= 70; y++ {
		chDesert.SetBlock(0, int16(y), 0, 0, stoneRID)
	}

	builder.BuildSurface(0, 0, chDesert)

	topDesert := chDesert.Block(0, 70, 0, 0)
	if topDesert != sandRID {
		t.Fatalf("expected sand at Y=70, got %d", topDesert)
	}
	subDesert := chDesert.Block(0, 65, 0, 0)
	if subDesert != sandstoneRID {
		t.Fatalf("expected sandstone at Y=65, got %d", subDesert)
	}
}

func BenchmarkBuildSurface(b *testing.B) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry
	stoneRID := reg.BlockRuntimeID(block.Stone{})

	ch := chunk.New(reg, cube.Range{-64, 319})
	for cx := range 16 {
		for cz := range 16 {
			for y := -64; y <= 70; y++ {
				ch.SetBlock(uint8(cx), int16(y), uint8(cz), 0, stoneRID)
			}
		}
	}

	builder := surface.NewRuleBuilder(12345)

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		builder.BuildSurface(i, i, ch)
	}
}

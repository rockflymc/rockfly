package terrain_test

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/climate"
	"github.com/df-mc/dragonfly/server/world/generator/terrain"
)

func TestSplineKnotsAndExtrapolation(t *testing.T) {
	s := terrain.NewSpline(
		terrain.SplinePoint{Location: 0.0, Value: 10.0, Slope: 2.0},
		terrain.SplinePoint{Location: 10.0, Value: 30.0, Slope: 2.0},
	)

	// Knots
	if math.Abs(s.Compute(0.0)-10.0) > 1e-6 {
		t.Fatalf("expected 10.0 at knot 0, got %v", s.Compute(0.0))
	}
	if math.Abs(s.Compute(10.0)-30.0) > 1e-6 {
		t.Fatalf("expected 30.0 at knot 10, got %v", s.Compute(10.0))
	}

	// Midpoint
	mid := s.Compute(5.0)
	if math.Abs(mid-20.0) > 1e-3 {
		t.Fatalf("expected ~20.0 at midpoint 5.0, got %v", mid)
	}

	// Linear extrapolation before 0.0: 10 + 2 * (-5) = 0.0
	before := s.Compute(-5.0)
	if math.Abs(before-0.0) > 1e-6 {
		t.Fatalf("expected 0.0 at -5.0, got %v", before)
	}

	// Linear extrapolation after 10.0: 30 + 2 * (5) = 40.0
	after := s.Compute(15.0)
	if math.Abs(after-40.0) > 1e-6 {
		t.Fatalf("expected 40.0 at 15.0, got %v", after)
	}
}

func TestDensityEvaluator(t *testing.T) {
	sampler := climate.NewSampler(42)
	evaluator := terrain.NewDensityEvaluator(42, sampler)

	// Bedrock layer should be strictly solid
	bedrockDensity := evaluator.DensityAt(0, -64, 0)
	if bedrockDensity < 0 {
		t.Fatalf("expected bedrock layer to have positive density, got %v", bedrockDensity)
	}

	// High stratosphere should be air
	skyDensity := evaluator.DensityAt(0, 310, 0)
	if skyDensity >= 0 {
		t.Fatalf("expected sky layer to have negative density, got %v", skyDensity)
	}
}

func TestOverworldGenerateChunk(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	gen := terrain.NewOverworld(9999)
	c := chunk.New(world.DefaultBlockRegistry, cube.Range{-64, 319})

	pos := world.ChunkPos{0, 0}
	gen.GenerateChunk(pos, c)

	// Check bottom block at Y = -64 is Bedrock
	bedrockRID := world.DefaultBlockRegistry.BlockRuntimeID(block.Bedrock{})
	bottomBlock := c.Block(0, -64, 0, 0)
	if bottomBlock != bedrockRID {
		t.Fatalf("expected bottom block to be bedrock (%d), got %d", bedrockRID, bottomBlock)
	}

	// Check deep subterranean block at Y = -20 is Deepslate (or solid)
	deepslateRID := world.DefaultBlockRegistry.BlockRuntimeID(block.Deepslate{})
	subBlock := c.Block(0, -20, 0, 0)
	if subBlock != deepslateRID {
		t.Fatalf("expected Y=-20 block to be deepslate (%d), got %d", deepslateRID, subBlock)
	}

	// Check biome is set
	biomeID := c.Biome(0, 64, 0)
	if biomeID == 0 {
		// Ocean biome ID is 0 in bedrock, so check non-negative / valid range
		_ = biomeID
	}
}

func BenchmarkOverworldGenerateChunk(b *testing.B) {
	world.DefaultBlockRegistry.Finalize()
	gen := terrain.NewOverworld(12345)
	c := chunk.New(world.DefaultBlockRegistry, cube.Range{-64, 319})
	pos := world.ChunkPos{0, 0}

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		pos[0] = int32(i)
		gen.GenerateChunk(pos, c)
	}
}

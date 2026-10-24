package feature

import (
	"math/rand/v2"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

func TestHeightDistributions(t *testing.T) {
	r := rand.New(rand.NewPCG(12345, 67890))

	// 1. Uniform distribution
	uniform := UniformDistribution{MinY: -64, MaxY: 320}
	for range 10000 {
		y := uniform.Sample(r)
		if y < -64 || y > 320 {
			t.Fatalf("Uniform out of bounds: %d", y)
		}
	}

	// 2. Triangle distribution peaking at Y=0
	triangle := TriangleDistribution{MinY: -32, MaxY: 32}
	var midCount, edgeCount int
	for range 10000 {
		y := triangle.Sample(r)
		if y < -32 || y > 32 {
			t.Fatalf("Triangle out of bounds: %d", y)
		}
		if y >= -8 && y <= 8 {
			midCount++
		}
		if y <= -24 || y >= 24 {
			edgeCount++
		}
	}
	if midCount <= edgeCount {
		t.Fatalf("Triangle distribution did not peak at center: mid=%d, edge=%d", midCount, edgeCount)
	}

	// 3. Inverted triangle distribution peaking at MinY
	inverted := InvertedTriangleDistribution{MinY: -64, MaxY: 16}
	var bottomCount, topCount int
	for range 10000 {
		y := inverted.Sample(r)
		if y < -64 || y > 16 {
			t.Fatalf("Inverted triangle out of bounds: %d", y)
		}
		if y <= -40 {
			bottomCount++
		}
		if y >= 0 {
			topCount++
		}
	}
	if bottomCount <= topCount {
		t.Fatalf("Inverted triangle did not skew towards bottom: bottom=%d, top=%d", bottomCount, topCount)
	}
}

func TestOreBlockReplacement(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry
	c := chunk.New(reg, cube.Range{-64, 319})

	stoneID := reg.BlockRuntimeID(block.Stone{})
	deepslateID := reg.BlockRuntimeID(block.Deepslate{})
	diamondStoneID := reg.BlockRuntimeID(block.DiamondOre{Type: block.StoneOre()})
	diamondDeepID := reg.BlockRuntimeID(block.DiamondOre{Type: block.DeepslateOre()})

	// Fill Y=10 with Stone, Y=-10 with Deepslate
	for x := range uint8(16) {
		for z := range uint8(16) {
			c.SetBlock(x, 10, z, 0, stoneID)
			c.SetBlock(x, -10, z, 0, deepslateID)
		}
	}

	placer := NewOrePlacerWithRegistry(reg)
	cfg := OreConfiguration{
		Name:              "test_diamond",
		StoneOreBlock:     diamondStoneID,
		DeepslateOreBlock: diamondDeepID,
		VeinSize:          4,
	}

	r := rand.New(rand.NewPCG(999, 888))
	// Place vein at Y=10 (Stone layer)
	placer.PlaceVein(c, 8, 10, 8, cfg, r)
	// Place vein at Y=-10 (Deepslate layer)
	placer.PlaceVein(c, 8, -10, 8, cfg, r)

	var foundStoneDiamond, foundDeepDiamond bool
	for x := range uint8(16) {
		for z := range uint8(16) {
			if c.Block(x, 10, z, 0) == diamondStoneID {
				foundStoneDiamond = true
			}
			if c.Block(x, -10, z, 0) == diamondDeepID {
				foundDeepDiamond = true
			}
		}
	}

	if !foundStoneDiamond {
		t.Fatalf("expected DiamondOre (Stone) to be placed at Y=10")
	}
	if !foundDeepDiamond {
		t.Fatalf("expected DiamondOre (Deepslate) to be placed at Y=-10")
	}
}

func TestAirExposureDiscard(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry
	c := chunk.New(reg, cube.Range{-64, 319})

	stoneID := reg.BlockRuntimeID(block.Stone{})
	airID := reg.BlockRuntimeID(block.Air{})
	lapisID := reg.BlockRuntimeID(block.LapisOre{Type: block.StoneOre()})

	// Set (8, 10, 8) to Stone, and adjacent (8, 11, 8) to Air
	c.SetBlock(8, 10, 8, 0, stoneID)
	c.SetBlock(8, 11, 8, 0, airID)

	placer := NewOrePlacerWithRegistry(reg)
	cfg := OreConfiguration{
		Name:               "test_buried_lapis",
		StoneOreBlock:      lapisID,
		VeinSize:           1,
		DiscardOnAirChance: 1.0, // 100% discard on air contact
	}

	r := rand.New(rand.NewPCG(111, 222))
	placer.PlaceVein(c, 8, 10, 8, cfg, r)

	// Block should remain Stone, not replaced by Lapis
	if c.Block(8, 10, 8, 0) == lapisID {
		t.Fatalf("buried ore was placed adjacent to air despite 100%% discard chance")
	}
}

func TestOverworldOrePopulation(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry
	c := chunk.New(reg, cube.Range{-64, 319})

	stoneID := reg.BlockRuntimeID(block.Stone{})
	deepslateID := reg.BlockRuntimeID(block.Deepslate{})
	diamondDeepID := reg.BlockRuntimeID(block.DiamondOre{Type: block.DeepslateOre()})
	coalStoneID := reg.BlockRuntimeID(block.CoalOre{Type: block.StoneOre()})

	// Fill realistic stone/deepslate column
	for x := range uint8(16) {
		for z := range uint8(16) {
			for y := int16(-64); y < 0; y++ {
				c.SetBlock(x, y, z, 0, deepslateID)
			}
			for y := int16(0); y < 200; y++ {
				c.SetBlock(x, y, z, 0, stoneID)
			}
		}
	}

	feat := NewOverworldOreFeature(reg)
	feat.Populate(c, 0, 0, 12345678)

	var diamondCount, coalCount int
	for x := range uint8(16) {
		for z := range uint8(16) {
			for y := int16(-64); y < 0; y++ {
				if c.Block(x, y, z, 0) == diamondDeepID {
					diamondCount++
				}
			}
			for y := int16(50); y < 190; y++ {
				if c.Block(x, y, z, 0) == coalStoneID {
					coalCount++
				}
			}
		}
	}

	if diamondCount == 0 {
		t.Fatalf("expected deepslate diamond ores to generate below Y=0")
	}
	if coalCount == 0 {
		t.Fatalf("expected coal ores to generate in positive Y stone")
	}
}

func TestNetherOrePopulation(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry
	c := chunk.New(reg, cube.Range{0, 127})

	netherrackID := reg.BlockRuntimeID(block.Netherrack{})
	quartzID := reg.BlockRuntimeID(block.NetherQuartzOre{})
	debrisID := reg.BlockRuntimeID(block.AncientDebris{})

	for x := range uint8(16) {
		for z := range uint8(16) {
			for y := int16(0); y < 128; y++ {
				c.SetBlock(x, y, z, 0, netherrackID)
			}
		}
	}

	feat := NewNetherOreFeature(reg)
	feat.Populate(c, 0, 0, 87654321)

	var quartzCount, debrisCount int
	for x := range uint8(16) {
		for z := range uint8(16) {
			for y := int16(10); y < 118; y++ {
				b := c.Block(x, y, z, 0)
				if b == quartzID {
					quartzCount++
				} else if b == debrisID {
					debrisCount++
				}
			}
		}
	}

	if quartzCount == 0 {
		t.Fatalf("expected nether quartz ores to generate in Nether")
	}
	if debrisCount == 0 {
		t.Fatalf("expected ancient debris to generate in Nether")
	}
}

func BenchmarkOverworldOrePopulate(b *testing.B) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry
	c := chunk.New(reg, cube.Range{-64, 319})
	stoneID := reg.BlockRuntimeID(block.Stone{})
	deepslateID := reg.BlockRuntimeID(block.Deepslate{})

	for x := range uint8(16) {
		for z := range uint8(16) {
			for y := int16(-64); y < 0; y++ {
				c.SetBlock(x, y, z, 0, deepslateID)
			}
			for y := int16(0); y < 120; y++ {
				c.SetBlock(x, y, z, 0, stoneID)
			}
		}
	}

	feat := NewOverworldOreFeature(reg)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		feat.Populate(c, 0, 0, int64(i))
	}
}

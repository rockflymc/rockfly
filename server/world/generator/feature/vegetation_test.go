package feature

import (
	"math/rand/v2"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

func init() {
	world.DefaultBlockRegistry.Finalize()
}

func TestVegetationPlacer_Trees(t *testing.T) {
	reg := world.DefaultBlockRegistry
	placer := NewVegetationPlacer(reg)
	grassID := reg.BlockRuntimeID(block.Grass{})
	r := rand.New(rand.NewPCG(42, 99))

	treeSpecies := []TreeType{
		TreeOak,
		TreeBirch,
		TreeSpruce,
		TreeAcacia,
		TreeDarkOak,
		TreeJungle,
		TreeCherry,
		TreePaleOak,
		TreeMangrove,
		TreePoplar,
	}

	for _, species := range treeSpecies {
		c := chunk.New(reg, cube.Range{-64, 319})
		// Set a grass surface block
		c.SetBlock(7, 64, 7, 0, grassID)

		ok := placer.PlaceTree(c, 7, 64, 7, species, r)
		if !ok {
			t.Fatalf("expected tree species %v to generate successfully", species)
		}

		// Verify trunk above surface
		trunkBlock := c.Block(7, 65, 7, 0)
		if trunkBlock == reg.AirRuntimeID() {
			t.Fatalf("expected trunk block at Y=65 for tree species %v", species)
		}
	}
}

func TestVegetationPlacer_DecorateChunk(t *testing.T) {
	reg := world.DefaultBlockRegistry
	placer := NewVegetationPlacer(reg)
	grassID := reg.BlockRuntimeID(block.Grass{})
	sandID := reg.BlockRuntimeID(block.Sand{})
	waterID := reg.BlockRuntimeID(block.Water{Still: true, Depth: 8})
	airID := reg.AirRuntimeID()

	c := chunk.New(reg, cube.Range{-64, 319})

	// Build grass surface in left half, sand surface in right half, water along border
	for x := range 16 {
		for z := range 16 {
			for y := int16(-64); y <= 64; y++ {
				switch {
				case x < 8:
					if y == 64 {
						c.SetBlock(uint8(x), y, uint8(z), 0, grassID)
					} else {
						c.SetBlock(uint8(x), y, uint8(z), 0, reg.BlockRuntimeID(block.Dirt{}))
					}
				case x == 8:
					c.SetBlock(uint8(x), y, uint8(z), 0, waterID)
				default:
					c.SetBlock(uint8(x), y, uint8(z), 0, sandID)
				}
			}
		}
	}

	placer.DecorateChunk(c, 10, 20, 1337)

	// Count surface flora above Y=64
	floraCount := 0
	for x := range 16 {
		for z := range 16 {
			if b := c.Block(uint8(x), 65, uint8(z), 0); b != airID {
				floraCount++
			}
		}
	}

	if floraCount == 0 {
		t.Fatalf("expected surface flora to be generated on grass and sand")
	}
}

func TestCaveFeaturePlacer_DecorateChunk(t *testing.T) {
	reg := world.DefaultBlockRegistry
	cavePlacer := NewCaveFeaturePlacer(reg)
	airID := reg.AirRuntimeID()
	stoneID := reg.BlockRuntimeID(block.Stone{})

	c := chunk.New(reg, cube.Range{-64, 319})

	// Create a subterranean cave between Y=-10 and Y=10 with stone ceiling & floor
	for x := range 16 {
		for z := range 16 {
			for y := int16(-40); y <= -11; y++ {
				c.SetBlock(uint8(x), y, uint8(z), 0, stoneID)
			}
			for y := int16(11); y <= 40; y++ {
				c.SetBlock(uint8(x), y, uint8(z), 0, stoneID)
			}
		}
	}

	cavePlacer.DecorateChunk(c, 5, 5, 9999)

	// Scan cave airspace for stalactites, stalagmites, moss carpet, spore blossoms
	placedFeatures := 0
	for x := range 16 {
		for z := range 16 {
			for y := int16(-10); y <= 10; y++ {
				if b := c.Block(uint8(x), y, uint8(z), 0); b != airID {
					placedFeatures++
				}
			}
		}
	}

	if placedFeatures == 0 {
		t.Fatalf("expected subterranean cave features (stalactites/stalagmites/moss) to be generated")
	}
}

func TestCaveFeaturePlacer_Geode(t *testing.T) {
	reg := world.DefaultBlockRegistry
	cavePlacer := NewCaveFeaturePlacer(reg)
	amethystID := reg.BlockRuntimeID(block.Amethyst{})
	calciteID := reg.BlockRuntimeID(block.Calcite{})
	smoothBasaltID := reg.BlockRuntimeID(block.SmoothBasalt{})

	c := chunk.New(reg, cube.Range{-64, 319})

	cavePlacer.generateGeode(c, 8, 0, 8, 4)

	foundAmethyst := false
	foundCalcite := false
	foundBasalt := false

	for x := range 16 {
		for z := range 16 {
			for y := int16(-5); y <= 5; y++ {
				switch b := c.Block(uint8(x), y, uint8(z), 0); b {
				case amethystID:
					foundAmethyst = true
				case calciteID:
					foundCalcite = true
				case smoothBasaltID:
					foundBasalt = true
				}
			}
		}
	}

	if !foundAmethyst || !foundCalcite || !foundBasalt {
		t.Fatalf("expected geode to contain amethyst, calcite, and smooth basalt layers")
	}
}

func TestVegetationNilSafety(t *testing.T) {
	var nilVeg *VegetationPlacer
	nilVeg.DecorateChunk(nil, 0, 0, 0)
	if nilVeg.PlaceTree(nil, 0, 0, 0, TreeOak, nil) {
		t.Fatal("expected false from nil tree placer")
	}

	var nilCave *CaveFeaturePlacer
	nilCave.DecorateChunk(nil, 0, 0, 0)
}

func BenchmarkVegetationPlacer_DecorateChunk(b *testing.B) {
	reg := world.DefaultBlockRegistry
	placer := NewVegetationPlacer(reg)
	grassID := reg.BlockRuntimeID(block.Grass{})

	c := chunk.New(reg, cube.Range{-64, 319})
	for x := range 16 {
		for z := range 16 {
			c.SetBlock(uint8(x), 64, uint8(z), 0, grassID)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		placer.DecorateChunk(c, i, i, 42)
	}
}

func BenchmarkCaveFeaturePlacer_DecorateChunk(b *testing.B) {
	reg := world.DefaultBlockRegistry
	cavePlacer := NewCaveFeaturePlacer(reg)
	stoneID := reg.BlockRuntimeID(block.Stone{})

	c := chunk.New(reg, cube.Range{-64, 319})
	for x := range 16 {
		for z := range 16 {
			c.SetBlock(uint8(x), -10, uint8(z), 0, stoneID)
			c.SetBlock(uint8(x), 10, uint8(z), 0, stoneID)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cavePlacer.DecorateChunk(c, i, i, 42)
	}
}

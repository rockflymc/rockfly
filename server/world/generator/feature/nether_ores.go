package feature

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// DefaultNetherOres initializes the full vanilla Nether mineral distribution suite.
func DefaultNetherOres(reg world.BlockRegistry) []OreConfiguration {
	quartzBlock := reg.BlockRuntimeID(block.NetherQuartzOre{})
	goldBlock := reg.BlockRuntimeID(block.NetherGoldOre{})
	debrisBlock := reg.BlockRuntimeID(block.AncientDebris{})

	return []OreConfiguration{
		// Nether Quartz: Uniform across Y: 10..117
		{
			Name:            "nether_quartz",
			NetherOreBlock:  quartzBlock,
			VeinSize:        14,
			BatchesPerChunk: 16,
			Distribution:    UniformDistribution{MinY: 10, MaxY: 117},
		},
		// Nether Gold: Uniform across Y: 10..117
		{
			Name:            "nether_gold",
			NetherOreBlock:  goldBlock,
			VeinSize:        10,
			BatchesPerChunk: 10,
			Distribution:    UniformDistribution{MinY: 10, MaxY: 117},
		},
		// Ancient Debris: Upper small batch (Y: 8..119, fully buried)
		{
			Name:               "ancient_debris_upper",
			NetherOreBlock:     debrisBlock,
			VeinSize:           2,
			BatchesPerChunk:    1,
			Distribution:       UniformDistribution{MinY: 8, MaxY: 119},
			DiscardOnAirChance: 1.0,
		},
		// Ancient Debris: Lower peak batch (Y: 8..24, peaking at Y=15, fully buried)
		{
			Name:               "ancient_debris_lower",
			NetherOreBlock:     debrisBlock,
			VeinSize:           3,
			BatchesPerChunk:    1,
			Distribution:       TriangleDistribution{MinY: 8, MaxY: 24},
			DiscardOnAirChance: 1.0,
		},
	}
}

// NetherOreFeature orchestrates ore placement across Nether chunk columns.
type NetherOreFeature struct {
	placer  *OrePlacer
	configs []OreConfiguration
}

// NewNetherOreFeature constructs a NetherOreFeature with default vanilla configs.
func NewNetherOreFeature(reg world.BlockRegistry) *NetherOreFeature {
	return &NetherOreFeature{
		placer:  NewOrePlacerWithRegistry(reg),
		configs: DefaultNetherOres(reg),
	}
}

// Populate scatters all Nether ores deterministically for chunk coordinates (chunkX, chunkZ).
func (f *NetherOreFeature) Populate(c *chunk.Chunk, chunkX, chunkZ int, seed int64) {
	chunkSeed := uint64(seed) ^ (uint64(chunkX)*0x4c1906) ^ (uint64(chunkZ)*0x5ac0db)
	r := rand.New(rand.NewPCG(chunkSeed, chunkSeed^0x9e3779b97f4a7c15))

	f.placer.PlaceChunk(c, chunkX, chunkZ, f.configs, r)
}

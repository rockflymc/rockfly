package feature

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// DefaultOverworldOres initializes the full 1.18+ vanilla Overworld mineral distribution suite.
func DefaultOverworldOres(reg world.BlockRegistry) []OreConfiguration {
	coalStone := reg.BlockRuntimeID(block.CoalOre{Type: block.StoneOre()})
	coalDeep := reg.BlockRuntimeID(block.CoalOre{Type: block.DeepslateOre()})

	ironStone := reg.BlockRuntimeID(block.IronOre{Type: block.StoneOre()})
	ironDeep := reg.BlockRuntimeID(block.IronOre{Type: block.DeepslateOre()})

	copperStone := reg.BlockRuntimeID(block.CopperOre{Type: block.StoneOre()})
	copperDeep := reg.BlockRuntimeID(block.CopperOre{Type: block.DeepslateOre()})

	goldStone := reg.BlockRuntimeID(block.GoldOre{Type: block.StoneOre()})
	goldDeep := reg.BlockRuntimeID(block.GoldOre{Type: block.DeepslateOre()})

	redstoneStone := reg.BlockRuntimeID(block.RedstoneOre{Type: block.StoneOre()})
	redstoneDeep := reg.BlockRuntimeID(block.RedstoneOre{Type: block.DeepslateOre()})

	lapisStone := reg.BlockRuntimeID(block.LapisOre{Type: block.StoneOre()})
	lapisDeep := reg.BlockRuntimeID(block.LapisOre{Type: block.DeepslateOre()})

	diamondStone := reg.BlockRuntimeID(block.DiamondOre{Type: block.StoneOre()})
	diamondDeep := reg.BlockRuntimeID(block.DiamondOre{Type: block.DeepslateOre()})

	emeraldStone := reg.BlockRuntimeID(block.EmeraldOre{Type: block.StoneOre()})
	emeraldDeep := reg.BlockRuntimeID(block.EmeraldOre{Type: block.DeepslateOre()})

	return []OreConfiguration{
		// Coal: Upper mountain triangle + Lower uniform
		{
			Name:              "coal_upper",
			StoneOreBlock:     coalStone,
			DeepslateOreBlock: coalDeep,
			VeinSize:          17,
			BatchesPerChunk:   20,
			Distribution:      TriangleDistribution{MinY: 136, MaxY: 320},
		},
		{
			Name:               "coal_lower",
			StoneOreBlock:      coalStone,
			DeepslateOreBlock:  coalDeep,
			VeinSize:           17,
			BatchesPerChunk:    15,
			Distribution:       UniformDistribution{MinY: 0, MaxY: 192},
			DiscardOnAirChance: 0.5,
		},
		// Iron: Mountain triangle, Middle triangle, Deep uniform
		{
			Name:              "iron_mountain",
			StoneOreBlock:     ironStone,
			DeepslateOreBlock: ironDeep,
			VeinSize:          9,
			BatchesPerChunk:   20,
			Distribution:      TriangleDistribution{MinY: 80, MaxY: 320},
		},
		{
			Name:              "iron_middle",
			StoneOreBlock:     ironStone,
			DeepslateOreBlock: ironDeep,
			VeinSize:          9,
			BatchesPerChunk:   10,
			Distribution:      TriangleDistribution{MinY: -24, MaxY: 56},
		},
		{
			Name:              "iron_small",
			StoneOreBlock:     ironStone,
			DeepslateOreBlock: ironDeep,
			VeinSize:          4,
			BatchesPerChunk:   10,
			Distribution:      UniformDistribution{MinY: -64, MaxY: -32},
		},
		// Copper: Triangle peaking at Y=48
		{
			Name:              "copper",
			StoneOreBlock:     copperStone,
			DeepslateOreBlock: copperDeep,
			VeinSize:          10,
			BatchesPerChunk:   16,
			Distribution:      TriangleDistribution{MinY: -16, MaxY: 112},
		},
		// Gold: Triangle peaking at Y=-16
		{
			Name:              "gold",
			StoneOreBlock:     goldStone,
			DeepslateOreBlock: goldDeep,
			VeinSize:          9,
			BatchesPerChunk:   4,
			Distribution:      TriangleDistribution{MinY: -64, MaxY: 32},
		},
		// Redstone: Uniform + Deep inverted triangle
		{
			Name:              "redstone_uniform",
			StoneOreBlock:     redstoneStone,
			DeepslateOreBlock: redstoneDeep,
			VeinSize:          8,
			BatchesPerChunk:   4,
			Distribution:      UniformDistribution{MinY: -64, MaxY: 15},
		},
		{
			Name:              "redstone_deep",
			StoneOreBlock:     redstoneStone,
			DeepslateOreBlock: redstoneDeep,
			VeinSize:          8,
			BatchesPerChunk:   8,
			Distribution:      InvertedTriangleDistribution{MinY: -64, MaxY: 15},
		},
		// Lapis Lazuli: Middle triangle + Buried uniform
		{
			Name:              "lapis",
			StoneOreBlock:     lapisStone,
			DeepslateOreBlock: lapisDeep,
			VeinSize:          7,
			BatchesPerChunk:   2,
			Distribution:      TriangleDistribution{MinY: -32, MaxY: 32},
		},
		{
			Name:               "lapis_buried",
			StoneOreBlock:      lapisStone,
			DeepslateOreBlock:  lapisDeep,
			VeinSize:           7,
			BatchesPerChunk:    4,
			Distribution:       UniformDistribution{MinY: -64, MaxY: 64},
			DiscardOnAirChance: 1.0,
		},
		// Diamond: Inverted triangle peaking at bottom Y=-64
		{
			Name:               "diamond_small",
			StoneOreBlock:      diamondStone,
			DeepslateOreBlock:  diamondDeep,
			VeinSize:           4,
			BatchesPerChunk:    7,
			Distribution:       InvertedTriangleDistribution{MinY: -64, MaxY: 16},
			DiscardOnAirChance: 0.5,
		},
		{
			Name:               "diamond_large_buried",
			StoneOreBlock:      diamondStone,
			DeepslateOreBlock:  diamondDeep,
			VeinSize:           8,
			BatchesPerChunk:    1,
			Distribution:       InvertedTriangleDistribution{MinY: -64, MaxY: 16},
			DiscardOnAirChance: 0.7,
		},
		// Emerald: High altitude single blocks
		{
			Name:              "emerald",
			StoneOreBlock:     emeraldStone,
			DeepslateOreBlock: emeraldDeep,
			VeinSize:          1,
			BatchesPerChunk:   10,
			Distribution:      TriangleDistribution{MinY: -16, MaxY: 320},
		},
	}
}

// OverworldOreFeature orchestrates ore placement across Overworld chunk columns.
type OverworldOreFeature struct {
	placer  *OrePlacer
	configs []OreConfiguration
}

// NewOverworldOreFeature constructs an OverworldOreFeature with default 1.18+ configs.
func NewOverworldOreFeature(reg world.BlockRegistry) *OverworldOreFeature {
	return &OverworldOreFeature{
		placer:  NewOrePlacerWithRegistry(reg),
		configs: DefaultOverworldOres(reg),
	}
}

// Populate scatters all Overworld ores deterministically for chunk coordinates (chunkX, chunkZ).
func (f *OverworldOreFeature) Populate(c *chunk.Chunk, chunkX, chunkZ int, seed int64) {
	// Deterministic chunk PRNG seeded with world seed + chunk coordinates
	chunkSeed := uint64(seed) ^ (uint64(chunkX)*0x4c1906) ^ (uint64(chunkZ)*0x5ac0db)
	r := rand.New(rand.NewPCG(chunkSeed, chunkSeed^0x9e3779b97f4a7c15))

	f.placer.PlaceChunk(c, chunkX, chunkZ, f.configs, r)
}

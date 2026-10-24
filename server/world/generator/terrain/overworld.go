package terrain

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/carver"
	"github.com/df-mc/dragonfly/server/world/generator/climate"
	"github.com/df-mc/dragonfly/server/world/generator/surface"
)

// Overworld implements world.Generator for authentic 1.18+ Bedrock Overworld generation.
type Overworld struct {
	seed    int64
	climate *climate.Sampler
	voronoi *climate.Voronoi
	density *DensityEvaluator
	carver  *carver.NoiseCarver
	surface *surface.RuleBuilder
	pool    sync.Pool

	stone     uint32
	deepslate uint32
	bedrock   uint32
	water     uint32
}

var _ world.Generator = (*Overworld)(nil)

// NewOverworld constructs an Overworld terrain generator from a world seed.
func NewOverworld(seed int64) *Overworld {
	return NewOverworldWithRegistry(seed, world.DefaultBlockRegistry)
}

// NewOverworldWithRegistry constructs an Overworld generator using a specific BlockRegistry.
func NewOverworldWithRegistry(seed int64, reg world.BlockRegistry) *Overworld {
	s := climate.NewSampler(seed)
	v := climate.NewVoronoi(s, seed)
	d := NewDensityEvaluator(seed, s)
	crv := carver.NewNoiseCarverWithRegistry(seed, reg)
	srf := surface.NewRuleBuilderWithRegistry(seed, reg)

	return &Overworld{
		seed:    seed,
		climate: s,
		voronoi: v,
		density: d,
		carver:  crv,
		surface: srf,
		pool: sync.Pool{
			New: func() any {
				return &CellGrid{}
			},
		},
		stone:     reg.BlockRuntimeID(block.Stone{}),
		deepslate: reg.BlockRuntimeID(block.Deepslate{}),
		bedrock:   reg.BlockRuntimeID(block.Bedrock{}),
		water:     reg.BlockRuntimeID(block.Water{Still: true, Depth: 8}),
	}
}

// GenerateChunk generates 3D terrain density and authentic biomes for the specified chunk column.
func (o *Overworld) GenerateChunk(pos world.ChunkPos, c *chunk.Chunk) {
	grid := o.pool.Get().(*CellGrid)
	defer o.pool.Put(grid)

	chunkX := int(pos.X())
	chunkZ := int(pos.Z())

	// Sample 3D density grid at 4x4x4 corner intervals
	o.density.SampleGrid(chunkX, chunkZ, grid)

	// Interpolate blocks into chunk column
	InterpolateChunk(c, grid, o.stone, o.deepslate, o.bedrock, o.water)

	// Carve subterranean cheese/spaghetti/noodle caves and aquifers
	o.carver.CarveChunk(chunkX, chunkZ, c)

	// Populate authentic biomes at quarter-resolution intervals
	originX := chunkX << 4
	originZ := chunkZ << 4

	for cx := range 4 {
		x := uint8(cx << 2)
		worldX := originX + int(x)

		for cz := range 4 {
			z := uint8(cz << 2)
			worldZ := originZ + int(z)

			for cy := range 96 {
				y := int16(-64 + (cy << 2))
				worldY := int(y)

				b := o.voronoi.BiomeAt(worldX, worldY, worldZ)
				c.SetBiome(x, y, z, uint32(b.EncodeBiome()))
			}
		}
	}

	// Apply rule-based surface builders and bedrock stratification
	o.surface.BuildSurface(chunkX, chunkZ, c)
}

// DefaultSpawn returns a reasonable world spawn position near the origin.
func (o *Overworld) DefaultSpawn(dim world.Dimension) cube.Pos {
	return cube.Pos{0, 64, 0}
}

// Seed returns the world generation seed.
func (o *Overworld) Seed() int64 {
	return o.seed
}

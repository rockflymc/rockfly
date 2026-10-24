package feature

import (
	"math"
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// OreConfiguration specifies placement parameters and block replacement targets for an ore vein.
type OreConfiguration struct {
	Name               string
	StoneOreBlock      uint32
	DeepslateOreBlock  uint32
	NetherOreBlock     uint32
	VeinSize           int
	BatchesPerChunk    int
	Distribution       Distribution
	DiscardOnAirChance float64
}

// OrePlacer evaluates and splats ore veins into generating chunk columns.
type OrePlacer struct {
	stone      uint32
	deepslate  uint32
	netherrack uint32
	air        uint32
}

// NewOrePlacer constructs an OrePlacer using world.DefaultBlockRegistry.
func NewOrePlacer() *OrePlacer {
	return NewOrePlacerWithRegistry(world.DefaultBlockRegistry)
}

// NewOrePlacerWithRegistry constructs an OrePlacer with a specified BlockRegistry.
func NewOrePlacerWithRegistry(reg world.BlockRegistry) *OrePlacer {
	return &OrePlacer{
		stone:      reg.BlockRuntimeID(block.Stone{}),
		deepslate:  reg.BlockRuntimeID(block.Deepslate{}),
		netherrack: reg.BlockRuntimeID(block.Netherrack{}),
		air:        reg.BlockRuntimeID(block.Air{}),
	}
}

// PlaceChunk processes all configured ore batches for a single chunk column.
func (p *OrePlacer) PlaceChunk(c *chunk.Chunk, chunkX, chunkZ int, configs []OreConfiguration, r *rand.Rand) {
	for _, cfg := range configs {
		for range cfg.BatchesPerChunk {
			originX := r.IntN(16)
			originZ := r.IntN(16)
			originY := cfg.Distribution.Sample(r)

			p.PlaceVein(c, originX, originY, originZ, cfg, r)
		}
	}
}

// PlaceVein generates a continuous ellipsoidal ore vein cluster.
func (p *OrePlacer) PlaceVein(c *chunk.Chunk, localX, y, localZ int, cfg OreConfiguration, r *rand.Rand) {
	if cfg.VeinSize <= 0 {
		return
	}
	if cfg.VeinSize == 1 {
		// Single block placement (e.g. Emerald ore)
		p.replaceBlock(c, uint8(localX), int16(y), uint8(localZ), cfg, r)
		return
	}

	n := float64(cfg.VeinSize)
	angle := r.Float64() * math.Pi
	dx := math.Sin(angle) * n / 8.0
	dz := math.Cos(angle) * n / 8.0

	x1 := float64(localX) + dx
	x2 := float64(localX) - dx
	z1 := float64(localZ) + dz
	z2 := float64(localZ) - dz

	y1 := float64(y + r.IntN(3) - 1)
	y2 := float64(y + r.IntN(3) - 1)

	steps := cfg.VeinSize
	for i := range steps {
		progress := float64(i) / n
		cx := x1 + (x2-x1)*progress
		cy := y1 + (y2-y1)*progress
		cz := z1 + (z2-z1)*progress

		sizeFactor := (r.Float64() * n / 16.0)
		radiusH := (math.Sin(math.Pi*progress)+1.0)*sizeFactor + 1.0
		radiusV := (math.Sin(math.Pi*progress)+1.0)*sizeFactor + 1.0

		minX := int(math.Floor(cx - radiusH))
		maxX := int(math.Ceil(cx + radiusH))
		minY := int(math.Floor(cy - radiusV))
		maxY := int(math.Ceil(cy + radiusV))
		minZ := int(math.Floor(cz - radiusH))
		maxZ := int(math.Ceil(cz + radiusH))

		radHSq := radiusH * radiusH
		radVSq := radiusV * radiusV

		for vx := minX; vx <= maxX; vx++ {
			if vx < 0 || vx >= 16 {
				continue
			}
			xDist := float64(vx) - cx
			xTerm := (xDist * xDist) / radHSq
			if xTerm > 1.0 {
				continue
			}

			for vz := minZ; vz <= maxZ; vz++ {
				if vz < 0 || vz >= 16 {
					continue
				}
				zDist := float64(vz) - cz
				zTerm := (zDist * zDist) / radHSq
				if xTerm+zTerm > 1.0 {
					continue
				}

				for vy := minY; vy <= maxY; vy++ {
					if vy < -64 || vy > 319 {
						continue
					}
					yDist := float64(vy) - cy
					yTerm := (yDist * yDist) / radVSq

					if xTerm+yTerm+zTerm <= 1.0 {
						p.replaceBlock(c, uint8(vx), int16(vy), uint8(vz), cfg, r)
					}
				}
			}
		}
	}
}

// replaceBlock inspects the target block and applies the appropriate ore variant.
func (p *OrePlacer) replaceBlock(c *chunk.Chunk, x uint8, y int16, z uint8, cfg OreConfiguration, r *rand.Rand) {
	current := c.Block(x, y, z, 0)

	var targetOre uint32
	switch current {
	case p.stone:
		targetOre = cfg.StoneOreBlock
	case p.deepslate:
		targetOre = cfg.DeepslateOreBlock
	case p.netherrack:
		targetOre = cfg.NetherOreBlock
	default:
		return
	}

	if targetOre == 0 {
		return
	}

	// Air exposure check: if adjacent voxel is air, apply air discard chance
	if cfg.DiscardOnAirChance > 0 {
		if p.isExposedToAir(c, x, y, z) {
			if r.Float64() < cfg.DiscardOnAirChance {
				return
			}
		}
	}

	c.SetBlock(x, y, z, 0, targetOre)
}

// isExposedToAir checks the 6 adjacent cardinal neighbors for air blocks.
func (p *OrePlacer) isExposedToAir(c *chunk.Chunk, x uint8, y int16, z uint8) bool {
	// Check Y boundaries
	if y+1 <= 319 && c.Block(x, y+1, z, 0) == p.air {
		return true
	}
	if y-1 >= -64 && c.Block(x, y-1, z, 0) == p.air {
		return true
	}

	// Check X neighbors within chunk
	if x > 0 && c.Block(x-1, y, z, 0) == p.air {
		return true
	}
	if x < 15 && c.Block(x+1, y, z, 0) == p.air {
		return true
	}

	// Check Z neighbors within chunk
	if z > 0 && c.Block(x, y, z-1, 0) == p.air {
		return true
	}
	if z < 15 && c.Block(x, y, z+1, 0) == p.air {
		return true
	}

	return false
}

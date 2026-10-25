package feature

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// CaveFeaturePlacer populates subterranean caves with dripstone stalactites/stalagmites,
// lush cave moss carpets, spore blossoms, amethyst geodes, and ceiling illumination.
type CaveFeaturePlacer struct {
	air          uint32
	stone        uint32
	deepslate    uint32
	dripstone    uint32
	mossCarpet   uint32
	sporeBlossom uint32
	glowstone    uint32
	calcite      uint32
	amethyst     uint32
	smoothBasalt uint32
	tuff         uint32
	cobweb       uint32
}

// NewCaveFeaturePlacer initializes CaveFeaturePlacer using world.BlockRegistry.
func NewCaveFeaturePlacer(reg world.BlockRegistry) *CaveFeaturePlacer {
	if reg == nil {
		reg = world.DefaultBlockRegistry
	}
	return &CaveFeaturePlacer{
		air:          reg.BlockRuntimeID(block.Air{}),
		stone:        reg.BlockRuntimeID(block.Stone{}),
		deepslate:    reg.BlockRuntimeID(block.Deepslate{}),
		dripstone:    reg.BlockRuntimeID(block.Dripstone{}),
		mossCarpet:   reg.BlockRuntimeID(block.MossCarpet{}),
		sporeBlossom: reg.BlockRuntimeID(block.SporeBlossom{}),
		glowstone:    reg.BlockRuntimeID(block.Glowstone{}),
		calcite:      reg.BlockRuntimeID(block.Calcite{}),
		amethyst:     reg.BlockRuntimeID(block.Amethyst{}),
		smoothBasalt: reg.BlockRuntimeID(block.SmoothBasalt{}),
		tuff:         reg.BlockRuntimeID(block.Tuff{}),
		cobweb:       reg.BlockRuntimeID(block.Cobweb{}),
	}
}

// DecorateChunk generates subterranean cave features in chunk (chunkX, chunkZ).
func (c *CaveFeaturePlacer) DecorateChunk(chk *chunk.Chunk, chunkX, chunkZ int, seed int64) {
	if c == nil || chk == nil {
		return
	}
	chunkSeed := uint64(seed) ^ (uint64(chunkX) * 0x85ebca6b) ^ (uint64(chunkZ) * 0xc2b2ae35)
	r := rand.New(rand.NewPCG(chunkSeed, chunkSeed^0x9e3779b97f4a7c15))

	// 1. Amethyst Geode Generation (roughly 1 in 20 chunks)
	if r.IntN(20) == 0 {
		gx := 4 + r.IntN(8)
		gz := 4 + r.IntN(8)
		gy := -45 + r.IntN(60)  // Y between -45 and 15
		radius := 3 + r.IntN(2) // radius 3 to 4
		c.generateGeode(chk, gx, gy, gz, radius)
	}

	// 2. Scan subterranean columns for cave surfaces (Y = -55 to 55)
	for lx := range 16 {
		for lz := range 16 {
			for y := int16(-55); y <= 55; y++ {
				curr := chk.Block(uint8(lx), y, uint8(lz), 0)
				if curr != c.air {
					continue
				}

				below := chk.Block(uint8(lx), y-1, uint8(lz), 0)
				above := chk.Block(uint8(lx), y+1, uint8(lz), 0)

				// Floor features
				if c.isSolidRock(below) {
					roll := r.Float64()
					switch {
					case roll < 0.04:
						// Lush cave moss carpet
						chk.SetBlock(uint8(lx), y, uint8(lz), 0, c.mossCarpet)
					case roll < 0.08:
						// Dripstone stalagmite (pointing upwards 1 to 3 blocks)
						height := 1 + r.IntN(3)
						for h := range height {
							py := y + int16(h)
							if py < 310 && chk.Block(uint8(lx), py, uint8(lz), 0) == c.air {
								chk.SetBlock(uint8(lx), py, uint8(lz), 0, c.dripstone)
							}
						}
					case roll < 0.085:
						// Cobweb in dark corner
						chk.SetBlock(uint8(lx), y, uint8(lz), 0, c.cobweb)
					}
				}

				// Ceiling features
				if c.isSolidRock(above) {
					roll := r.Float64()
					switch {
					case roll < 0.02:
						// Spore blossom hanging from ceiling
						chk.SetBlock(uint8(lx), y, uint8(lz), 0, c.sporeBlossom)
					case roll < 0.06:
						// Dripstone stalactite (hanging downwards 1 to 4 blocks)
						height := 1 + r.IntN(4)
						for h := range height {
							py := y - int16(h)
							if py > -64 && chk.Block(uint8(lx), py, uint8(lz), 0) == c.air {
								chk.SetBlock(uint8(lx), py, uint8(lz), 0, c.dripstone)
							}
						}
					case roll < 0.075:
						// Glowstone ceiling cluster in lush/deep cave
						chk.SetBlock(uint8(lx), y+1, uint8(lz), 0, c.glowstone)
					}
				}
			}
		}
	}
}

// isSolidRock returns true if the block runtime ID matches stone or deepslate.
func (c *CaveFeaturePlacer) isSolidRock(b uint32) bool {
	return b == c.stone || b == c.deepslate || b == c.tuff
}

// generateGeode carves and places a concentric spherical amethyst geode into the chunk.
func (c *CaveFeaturePlacer) generateGeode(chk *chunk.Chunk, cx, cy, cz, radius int) {
	if c == nil || chk == nil {
		return
	}
	rOuterSq := radius * radius
	rMidSq := (radius - 1) * (radius - 1)
	rInnerSq := (radius - 2) * (radius - 2)

	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			for dz := -radius; dz <= radius; dz++ {
				x := cx + dx
				y := cy + dy
				z := cz + dz

				if x < 0 || x >= 16 || z < 0 || z >= 16 || y < -60 || y >= 310 {
					continue
				}

				distSq := dx*dx + dy*dy + dz*dz
				if distSq > rOuterSq {
					continue
				}

				switch {
				case distSq <= rInnerSq:
					// Hollow cave interior
					chk.SetBlock(uint8(x), int16(y), uint8(z), 0, c.air)
				case distSq <= rMidSq:
					// Amethyst crystal block lining
					chk.SetBlock(uint8(x), int16(y), uint8(z), 0, c.amethyst)
				default:
					// Outer shells: calcite and smooth basalt
					if distSq%2 == 0 {
						chk.SetBlock(uint8(x), int16(y), uint8(z), 0, c.calcite)
					} else {
						chk.SetBlock(uint8(x), int16(y), uint8(z), 0, c.smoothBasalt)
					}
				}
			}
		}
	}
}

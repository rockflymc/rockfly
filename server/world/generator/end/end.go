package end

import (
	"math"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/noise"
)

// End implements world.Generator for authentic 3D End dimension generation.
type End struct {
	seed     int64
	noise3D  *noise.OctaveNoise
	endStone uint32
	endBiome uint32
}

var _ world.Generator = (*End)(nil)

// NewEnd constructs an End dimension generator with the world seed.
func NewEnd(seed int64) *End {
	return NewEndWithRegistry(seed, world.DefaultBlockRegistry)
}

// NewEndWithRegistry constructs an End dimension generator using a specific BlockRegistry.
func NewEndWithRegistry(seed int64, reg world.BlockRegistry) *End {
	return &End{
		seed:     seed,
		noise3D:  noise.NewOctavePerlinFromSeed(seed^0x456e6449, 3, 2.0, 0.5), // "EndI"
		endStone: reg.BlockRuntimeID(block.EndStone{}),
		endBiome: uint32(biome.End{}.EncodeBiome()),
	}
}

// GenerateChunk generates the Central Island, void ring, and outer floating End islands.
func (e *End) GenerateChunk(pos world.ChunkPos, c *chunk.Chunk) {
	originX := int(pos.X()) << 4
	originZ := int(pos.Z()) << 4

	// Populate End biome across the chunk column
	for cx := range 4 {
		x := uint8(cx << 2)
		for cz := range 4 {
			z := uint8(cz << 2)
			for cy := range 64 {
				c.SetBiome(x, int16(cy<<2), z, e.endBiome)
			}
		}
	}

	for cx := range 16 {
		x := uint8(cx)
		wx := originX + cx

		for cz := range 16 {
			z := uint8(cz)
			wz := originZ + cz

			distSq := float64(wx*wx + wz*wz)
			dist := math.Sqrt(distSq)

			switch {
			case dist <= 120.0:
				// 1. Central End Island: Parabolic dome with organic noisy underside
				// Relative radial distance [0.0 .. 1.0]
				relR := dist / 120.0
				thickness := (1.0 - relR*relR) * 25.0 // Max 25 blocks half-height

				minY := int(60.0 - thickness)
				maxY := int(60.0 + thickness*0.6)

				for y := minY; y <= maxY; y++ {
					// 3D noise variation along the perimeter and bottom
					noiseSample := e.noise3D.Sample3D(float64(wx)*0.04, float64(y)*0.04, float64(wz)*0.04) * 4.0
					if float64(y) >= float64(minY)+noiseSample && float64(y) <= float64(maxY)+noiseSample {
						c.SetBlock(x, int16(y), z, 0, e.endStone)
					}
				}

			case dist < 1000.0:
				// 2. Void gap between Central Island and Outer Islands: 100% empty space

			default:
				// 3. Outer End Islands: 3D floating landmasses
				for y := 45; y <= 75; y++ {
					noiseVal := e.noise3D.Sample3D(float64(wx)*0.02, float64(y)*0.04, float64(wz)*0.02)
					// Vertical falloff dampening
					falloff := math.Abs(float64(y-60)) / 15.0
					if noiseVal-falloff > 0.15 {
						c.SetBlock(x, int16(y), z, 0, e.endStone)
					}
				}
			}
		}
	}
}

// DefaultSpawn returns the canonical obsidian platform position in the End.
func (e *End) DefaultSpawn(dim world.Dimension) cube.Pos {
	return cube.Pos{100, 49, 0}
}

package carver

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// NoiseCarver orchestrates cheese, spaghetti, and noodle cave carving alongside subterranean aquifers.
type NoiseCarver struct {
	cheese    *CheeseCarver
	spaghetti *SpaghettiCarver
	noodle    *NoodleCarver
	aquifer   *AquiferSampler

	stone     uint32
	deepslate uint32
	bedrock   uint32
	air       uint32
}

// NewNoiseCarver constructs a NoiseCarver for the given world seed using DefaultBlockRegistry.
func NewNoiseCarver(seed int64) *NoiseCarver {
	return NewNoiseCarverWithRegistry(seed, world.DefaultBlockRegistry)
}

// NewNoiseCarverWithRegistry constructs a NoiseCarver using a specific world.BlockRegistry.
func NewNoiseCarverWithRegistry(seed int64, reg world.BlockRegistry) *NoiseCarver {
	air := reg.BlockRuntimeID(block.Air{})
	water := reg.BlockRuntimeID(block.Water{Still: true, Depth: 8})
	lava := reg.BlockRuntimeID(block.Lava{Still: true, Depth: 8})
	stone := reg.BlockRuntimeID(block.Stone{})
	deepslate := reg.BlockRuntimeID(block.Deepslate{})
	bedrock := reg.BlockRuntimeID(block.Bedrock{})

	return &NoiseCarver{
		cheese:    NewCheeseCarver(seed),
		spaghetti: NewSpaghettiCarver(seed),
		noodle:    NewNoodleCarver(seed),
		aquifer:   NewAquiferSampler(seed, air, water, lava),
		stone:     stone,
		deepslate: deepslate,
		bedrock:   bedrock,
		air:       air,
	}
}

// ShouldCarve returns true if any cave generator carves at the given voxel coordinate.
func (c *NoiseCarver) ShouldCarve(x, y, z int) bool {
	return c.cheese.IsCave(x, y, z) || c.spaghetti.IsCave(x, y, z) || c.noodle.IsCave(x, y, z)
}

// CarveChunk carves subterranean caverns and fills subterranean aquifers across a chunk column.
func (c *NoiseCarver) CarveChunk(chunkX, chunkZ int, ch *chunk.Chunk) {
	originX := chunkX << 4
	originZ := chunkZ << 4

	for cx := range 16 {
		x := uint8(cx)
		wx := originX + cx

		for cz := range 16 {
			z := uint8(cz)
			wz := originZ + cz

			for y := -63; y <= 62; y++ {
				current := ch.Block(x, int16(y), z, 0)
				// Only carve through solid stone and deepslate; preserve bedrock at y=-64.
				if current != c.stone && current != c.deepslate {
					continue
				}

				if c.ShouldCarve(wx, y, wz) {
					fluid := c.aquifer.FluidAt(wx, y, wz)
					ch.SetBlock(x, int16(y), z, 0, fluid)
				}
			}
		}
	}
}

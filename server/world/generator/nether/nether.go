package nether

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/noise"
)

// Nether implements world.Generator for authentic 3D Nether dimension generation.
type Nether struct {
	seed       int64
	noise3D    *noise.OctaveNoise
	biomeNoise *noise.OctaveNoise

	netherrack uint32
	lava       uint32
	bedrock    uint32
	soulSand   uint32
	basalt     uint32
	blackstone uint32

	wastesID     uint32
	crimsonID    uint32
	warpedID     uint32
	soulValleyID uint32
	deltasID     uint32
}

var _ world.Generator = (*Nether)(nil)

// NewNether constructs a Nether dimension generator with the world seed.
func NewNether(seed int64) *Nether {
	return NewNetherWithRegistry(seed, world.DefaultBlockRegistry)
}

// NewNetherWithRegistry constructs a Nether dimension generator using a specific BlockRegistry.
func NewNetherWithRegistry(seed int64, reg world.BlockRegistry) *Nether {
	return &Nether{
		seed:       seed,
		noise3D:    noise.NewOctavePerlinFromSeed(seed^0x4e657468, 4, 2.0, 0.5), // "Neth"
		biomeNoise: noise.NewOctavePerlinFromSeed(seed^0x42696f6d, 2, 2.0, 0.5), // "Biom"

		netherrack: reg.BlockRuntimeID(block.Netherrack{}),
		lava:       reg.BlockRuntimeID(block.Lava{Still: true, Depth: 8}),
		bedrock:    reg.BlockRuntimeID(block.Bedrock{}),
		soulSand:   reg.BlockRuntimeID(block.SoulSand{}),
		basalt:     reg.BlockRuntimeID(block.Basalt{}),
		blackstone: reg.BlockRuntimeID(block.Blackstone{}),

		wastesID:     uint32(biome.NetherWastes{}.EncodeBiome()),
		crimsonID:    uint32(biome.CrimsonForest{}.EncodeBiome()),
		warpedID:     uint32(biome.WarpedForest{}.EncodeBiome()),
		soulValleyID: uint32(biome.SoulSandValley{}.EncodeBiome()),
		deltasID:     uint32(biome.BasaltDeltas{}.EncodeBiome()),
	}
}

// GenerateChunk populates dual bedrock ceilings/floors, 3D Netherrack cavern density,
// the Y=31 lava ocean, and authentic Nether biomes across the chunk column.
func (n *Nether) GenerateChunk(pos world.ChunkPos, c *chunk.Chunk) {
	originX := int(pos.X()) << 4
	originZ := int(pos.Z()) << 4

	for cx := range 16 {
		x := uint8(cx)
		wx := originX + cx

		for cz := range 16 {
			z := uint8(cz)
			wz := originZ + cz

			// Resolve Nether biome from 2D continuous noise
			bVal := n.biomeNoise.SampleNormalized2D(float64(wx)*0.005, float64(wz)*0.005)
			var bID uint32
			switch {
			case bVal < -0.4:
				bID = n.soulValleyID
			case bVal < -0.1:
				bID = n.crimsonID
			case bVal < 0.2:
				bID = n.wastesID
			case bVal < 0.5:
				bID = n.warpedID
			default:
				bID = n.deltasID
			}

			// Assign biome across quarter-resolution intervals
			if cx%4 == 0 && cz%4 == 0 {
				for cy := range 32 { // 0..127 in steps of 4
					c.SetBiome(x, int16(cy<<2), z, bID)
				}
			}

			// Generate vertical volume (Y: 0 to 127)
			for y := 0; y <= 127; y++ {
				// 1. Dual bedrock boundaries (floor Y in [0, 4], ceiling Y in [123, 127])
				if y == 0 || y == 127 {
					c.SetBlock(x, int16(y), z, 0, n.bedrock)
					continue
				}
				if y <= 4 {
					if n.dither(wx, y, wz, 5-y) {
						c.SetBlock(x, int16(y), z, 0, n.bedrock)
						continue
					}
				}
				if y >= 123 {
					if n.dither(wx, y, wz, y-122) {
						c.SetBlock(x, int16(y), z, 0, n.bedrock)
						continue
					}
				}

				// 2. 3D Netherrack cavern volumetric density
				var baseDensity float64
				switch {
				case y < 10:
					baseDensity = float64(10-y) * 0.2
				case y > 115:
					baseDensity = float64(y-115) * 0.2
				default:
					baseDensity = -0.25
				}

				density := n.noise3D.Sample3D(float64(wx)*0.03, float64(y)*0.04, float64(wz)*0.03) + baseDensity

				if density >= 0 {
					// Solid nether block
					switch {
					case bID == n.soulValleyID && y <= 35:
						c.SetBlock(x, int16(y), z, 0, n.soulSand)
					case bID == n.deltasID && (y%2 == 0):
						c.SetBlock(x, int16(y), z, 0, n.basalt)
					case bID == n.deltasID:
						c.SetBlock(x, int16(y), z, 0, n.blackstone)
					default:
						c.SetBlock(x, int16(y), z, 0, n.netherrack)
					}
				} else if y <= 31 {
					// 3. Lava ocean filling cavities at Y <= 31
					c.SetBlock(x, int16(y), z, 0, n.lava)
				}
			}
		}
	}
}

// dither returns deterministic boolean for bedrock stratification dithering.
func (n *Nether) dither(x, y, z, weight int) bool {
	h := int64(x)*3129871 ^ int64(z)*116129781 ^ int64(y)*2654435761 ^ n.seed
	h = (h ^ (h >> 16)) * 0x45d9f3b
	h = (h ^ (h >> 16)) & 0x7fffffff
	return (h % 5) < int64(weight)
}

// DefaultSpawn returns a reasonable spawn point in the Nether.
func (n *Nether) DefaultSpawn(dim world.Dimension) cube.Pos {
	return cube.Pos{0, 64, 0}
}

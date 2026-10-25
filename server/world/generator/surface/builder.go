package surface

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// RuleBuilder manages rule-based biome surface layer generation and bedrock stratification.
type RuleBuilder struct {
	seed     int64
	badlands *BadlandsBands

	grass     uint32
	dirt      uint32
	sand      uint32
	redSand   uint32
	sandstone uint32
	gravel    uint32
	podzol    uint32
	snow      uint32
	stone     uint32
	deepslate uint32
	bedrock   uint32
	water     uint32

	desertID         uint32
	badlandsID       uint32
	erodedBadlandsID uint32
	woodedBadlandsID uint32
	snowyID          uint32
	snowyBeachID     uint32
	snowyTaigaID     uint32
	snowySlopesID    uint32
	iceSpikesID      uint32
	frozenPeaksID    uint32
	taigaID          uint32
	oldGrowthPineID  uint32
	oldGrowthSpruceID uint32
	beachID          uint32
	stonyID          uint32
}

// NewRuleBuilder constructs a RuleBuilder using the DefaultBlockRegistry and given seed.
func NewRuleBuilder(seed int64) *RuleBuilder {
	return NewRuleBuilderWithRegistry(seed, world.DefaultBlockRegistry)
}

// NewRuleBuilderWithRegistry constructs a RuleBuilder using a specific BlockRegistry.
func NewRuleBuilderWithRegistry(seed int64, reg world.BlockRegistry) *RuleBuilder {
	if reg == nil {
		reg = world.DefaultBlockRegistry
	}
	return &RuleBuilder{
		seed:      seed,
		badlands:  NewBadlandsBands(reg),
		grass:     reg.BlockRuntimeID(block.Grass{}),
		dirt:      reg.BlockRuntimeID(block.Dirt{}),
		sand:      reg.BlockRuntimeID(block.Sand{Red: false}),
		redSand:   reg.BlockRuntimeID(block.Sand{Red: true}),
		sandstone: reg.BlockRuntimeID(block.Sandstone{}),
		gravel:    reg.BlockRuntimeID(block.Gravel{}),
		podzol:    reg.BlockRuntimeID(block.Podzol{}),
		snow:      reg.BlockRuntimeID(block.Snow{}),
		stone:     reg.BlockRuntimeID(block.Stone{}),
		deepslate: reg.BlockRuntimeID(block.Deepslate{}),
		bedrock:   reg.BlockRuntimeID(block.Bedrock{}),
		water:     reg.BlockRuntimeID(block.Water{Still: true, Depth: 8}),

		desertID:          uint32(biome.Desert{}.EncodeBiome()),
		badlandsID:        uint32(biome.Badlands{}.EncodeBiome()),
		erodedBadlandsID:  uint32(biome.ErodedBadlands{}.EncodeBiome()),
		woodedBadlandsID:  uint32(biome.WoodedBadlandsPlateau{}.EncodeBiome()),
		snowyID:           uint32(biome.SnowyPlains{}.EncodeBiome()),
		snowyBeachID:      uint32(biome.SnowyBeach{}.EncodeBiome()),
		snowyTaigaID:      uint32(biome.SnowyTaiga{}.EncodeBiome()),
		snowySlopesID:     uint32(biome.SnowySlopes{}.EncodeBiome()),
		iceSpikesID:       uint32(biome.IceSpikes{}.EncodeBiome()),
		frozenPeaksID:     uint32(biome.FrozenPeaks{}.EncodeBiome()),
		taigaID:           uint32(biome.Taiga{}.EncodeBiome()),
		oldGrowthPineID:   uint32(biome.OldGrowthPineTaiga{}.EncodeBiome()),
		oldGrowthSpruceID: uint32(biome.OldGrowthSpruceTaiga{}.EncodeBiome()),
		beachID:           uint32(biome.Beach{}.EncodeBiome()),
		stonyID:           uint32(biome.StonyPeaks{}.EncodeBiome()),
	}
}

// BuildSurface populates biome surface blocks and authentic bedrock stratification for a chunk.
func (b *RuleBuilder) BuildSurface(chunkX, chunkZ int, c *chunk.Chunk) {
	if b == nil || c == nil {
		return
	}
	originX := chunkX << 4
	originZ := chunkZ << 4

	for cx := range 16 {
		x := uint8(cx)
		wx := originX + cx

		for cz := range 16 {
			z := uint8(cz)
			wz := originZ + cz

			// 1. Bedrock floor stratification at Y in [-64, -60]
			for y := -64; y <= -60; y++ {
				if StratifyBedrock(wx, y, wz, b.seed) {
					c.SetBlock(x, int16(y), z, 0, b.bedrock)
				} else if y > -64 && c.Block(x, int16(y), z, 0) == b.bedrock {
					c.SetBlock(x, int16(y), z, 0, b.deepslate)
				}
			}

			// 2. Subterranean stone -> deepslate dithering at Y in [0, 8]
			for y := 0; y <= 8; y++ {
				if c.Block(x, int16(y), z, 0) == b.stone && DeepslateDither(wx, y, wz, b.seed) {
					c.SetBlock(x, int16(y), z, 0, b.deepslate)
				}
			}

			// 3. Biome surface layers
			biomeID := c.Biome(x, 64, z)

			// Scan down from ceiling to find upper solid boundary
			surfaceY := -100
			for y := 319; y >= -50; y-- {
				curr := c.Block(x, int16(y), z, 0)
				if curr == b.stone || curr == b.deepslate {
					surfaceY = y
					break
				}
			}

			if surfaceY <= -50 {
				continue // Void column or completely air/water
			}

			// Apply biome-appropriate surface rule
			if surfaceY < 62 {
				// Underwater ocean floor
				for d := 0; d < 3 && surfaceY-d >= -55; d++ {
					targetY := int16(surfaceY - d)
					if c.Block(x, targetY, z, 0) == b.stone || c.Block(x, targetY, z, 0) == b.deepslate {
						c.SetBlock(x, targetY, z, 0, b.gravel)
					}
				}
			} else {
				// Dry terrain surface above sea level
				switch biomeID {
				case b.desertID, b.beachID:
					for d := 0; d < 4 && surfaceY-d >= -55; d++ {
						c.SetBlock(x, int16(surfaceY-d), z, 0, b.sand)
					}
					for d := 4; d < 7 && surfaceY-d >= -55; d++ {
						c.SetBlock(x, int16(surfaceY-d), z, 0, b.sandstone)
					}
				case b.badlandsID, b.erodedBadlandsID, b.woodedBadlandsID:
					c.SetBlock(x, int16(surfaceY), z, 0, b.redSand)
					for d := 1; d < 20 && surfaceY-d >= -55; d++ {
						ty := surfaceY - d
						c.SetBlock(x, int16(ty), z, 0, b.badlands.BandAt(ty))
					}
				case b.snowyID, b.snowyBeachID, b.snowyTaigaID, b.snowySlopesID, b.iceSpikesID, b.frozenPeaksID:
					c.SetBlock(x, int16(surfaceY), z, 0, b.snow)
					for d := 1; d < 4 && surfaceY-d >= -55; d++ {
						c.SetBlock(x, int16(surfaceY-d), z, 0, b.dirt)
					}
				case b.taigaID, b.oldGrowthPineID, b.oldGrowthSpruceID:
					c.SetBlock(x, int16(surfaceY), z, 0, b.podzol)
					for d := 1; d < 4 && surfaceY-d >= -55; d++ {
						c.SetBlock(x, int16(surfaceY-d), z, 0, b.dirt)
					}
				case b.stonyID:
					// Stony peaks remain stone
				default:
					// Default plains / forest: grass block on top, dirt underneath
					c.SetBlock(x, int16(surfaceY), z, 0, b.grass)
					for d := 1; d < 4 && surfaceY-d >= -55; d++ {
						c.SetBlock(x, int16(surfaceY-d), z, 0, b.dirt)
					}
				}
			}
		}
	}
}

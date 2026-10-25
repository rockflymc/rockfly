package feature

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// TreeType defines the species of tree to generate.
type TreeType int

const (
	TreeOak TreeType = iota
	TreeBirch
	TreeSpruce
	TreeAcacia
	TreeDarkOak
	TreeJungle
	TreeCherry
	TreePaleOak
	TreeMangrove
	TreePoplar
)

// VegetationPlacer generates trees, grass, flowers, and surface vegetation directly into chunks.
type VegetationPlacer struct {
	air   uint32
	grass uint32
	dirt  uint32
	sand  uint32
	water uint32

	// Tree materials for all 10 species
	oakLog, oakLeaves           uint32
	birchLog, birchLeaves       uint32
	spruceLog, spruceLeaves     uint32
	acaciaLog, acaciaLeaves     uint32
	darkOakLog, darkOakLeaves   uint32
	jungleLog, jungleLeaves     uint32
	cherryLog, cherryLeaves     uint32
	paleOakLog, paleOakLeaves   uint32
	mangroveLog, mangroveLeaves uint32
	poplarLog, poplarLeaves     uint32

	// Surface flora
	shortGrass uint32
	fern       uint32
	dandelion  uint32
	poppy      uint32
	cornflower uint32
	allium     uint32
	azureBluet uint32
	blueOrchid uint32
	redTulip   uint32
	whiteTulip uint32
	pinkTulip  uint32
	oxeyeDaisy uint32
	cactus     uint32
	sugarCane  uint32
	pumpkin    uint32
	melon      uint32

	// Double flowers (upper and lower halves)
	sunflowerBottom, sunflowerTop uint32
	lilacBottom, lilacTop         uint32
	roseBottom, roseTop           uint32
	peonyBottom, peonyTop         uint32
}

// NewVegetationPlacer initializes a VegetationPlacer using a BlockRegistry.
func NewVegetationPlacer(reg world.BlockRegistry) *VegetationPlacer {
	if reg == nil {
		reg = world.DefaultBlockRegistry
	}
	return &VegetationPlacer{
		air:   reg.BlockRuntimeID(block.Air{}),
		grass: reg.BlockRuntimeID(block.Grass{}),
		dirt:  reg.BlockRuntimeID(block.Dirt{}),
		sand:  reg.BlockRuntimeID(block.Sand{}),
		water: reg.BlockRuntimeID(block.Water{Still: true, Depth: 8}),

		oakLog:    reg.BlockRuntimeID(block.Log{Wood: block.OakWood()}),
		oakLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.OakLeaves()}),

		birchLog:    reg.BlockRuntimeID(block.Log{Wood: block.BirchWood()}),
		birchLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.BirchLeaves()}),

		spruceLog:    reg.BlockRuntimeID(block.Log{Wood: block.SpruceWood()}),
		spruceLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.SpruceLeaves()}),

		acaciaLog:    reg.BlockRuntimeID(block.Log{Wood: block.AcaciaWood()}),
		acaciaLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.AcaciaLeaves()}),

		darkOakLog:    reg.BlockRuntimeID(block.Log{Wood: block.DarkOakWood()}),
		darkOakLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.DarkOakLeaves()}),

		jungleLog:    reg.BlockRuntimeID(block.Log{Wood: block.JungleWood()}),
		jungleLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.JungleLeaves()}),

		cherryLog:    reg.BlockRuntimeID(block.Log{Wood: block.CherryWood()}),
		cherryLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.CherryLeaves()}),

		paleOakLog:    reg.BlockRuntimeID(block.Log{Wood: block.PaleOakWood()}),
		paleOakLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.PaleOakLeaves()}),

		mangroveLog:    reg.BlockRuntimeID(block.Log{Wood: block.MangroveWood()}),
		mangroveLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.MangroveLeaves()}),

		poplarLog:    reg.BlockRuntimeID(block.Log{Wood: block.PoplarWood()}),
		poplarLeaves: reg.BlockRuntimeID(block.Leaves{Type: block.OrangePoplarLeaves()}),

		shortGrass: reg.BlockRuntimeID(block.ShortGrass{}),
		fern:       reg.BlockRuntimeID(block.Fern{}),
		dandelion:  reg.BlockRuntimeID(block.Flower{Type: block.Dandelion()}),
		poppy:      reg.BlockRuntimeID(block.Flower{Type: block.Poppy()}),
		cornflower: reg.BlockRuntimeID(block.Flower{Type: block.Cornflower()}),
		allium:     reg.BlockRuntimeID(block.Flower{Type: block.Allium()}),
		azureBluet: reg.BlockRuntimeID(block.Flower{Type: block.AzureBluet()}),
		blueOrchid: reg.BlockRuntimeID(block.Flower{Type: block.BlueOrchid()}),
		redTulip:   reg.BlockRuntimeID(block.Flower{Type: block.RedTulip()}),
		whiteTulip: reg.BlockRuntimeID(block.Flower{Type: block.WhiteTulip()}),
		pinkTulip:  reg.BlockRuntimeID(block.Flower{Type: block.PinkTulip()}),
		oxeyeDaisy: reg.BlockRuntimeID(block.Flower{Type: block.OxeyeDaisy()}),
		cactus:     reg.BlockRuntimeID(block.Cactus{}),
		sugarCane:  reg.BlockRuntimeID(block.SugarCane{}),
		pumpkin:    reg.BlockRuntimeID(block.Pumpkin{}),
		melon:      reg.BlockRuntimeID(block.Melon{}),

		sunflowerBottom: reg.BlockRuntimeID(block.DoubleFlower{Type: block.Sunflower(), UpperPart: false}),
		sunflowerTop:    reg.BlockRuntimeID(block.DoubleFlower{Type: block.Sunflower(), UpperPart: true}),
		lilacBottom:     reg.BlockRuntimeID(block.DoubleFlower{Type: block.Lilac(), UpperPart: false}),
		lilacTop:        reg.BlockRuntimeID(block.DoubleFlower{Type: block.Lilac(), UpperPart: true}),
		roseBottom:      reg.BlockRuntimeID(block.DoubleFlower{Type: block.RoseBush(), UpperPart: false}),
		roseTop:         reg.BlockRuntimeID(block.DoubleFlower{Type: block.RoseBush(), UpperPart: true}),
		peonyBottom:     reg.BlockRuntimeID(block.DoubleFlower{Type: block.Peony(), UpperPart: false}),
		peonyTop:        reg.BlockRuntimeID(block.DoubleFlower{Type: block.Peony(), UpperPart: true}),
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// PlaceTree splats a species-specific tree at local coordinates (lx, lz) atop surfaceY.
func (p *VegetationPlacer) PlaceTree(c *chunk.Chunk, lx, surfaceY, lz int, treeType TreeType, r *rand.Rand) bool {
	if p == nil || c == nil || r == nil {
		return false
	}
	if lx < 2 || lx > 13 || lz < 2 || lz > 13 || surfaceY < -50 || surfaceY > 300 {
		return false
	}

	ground := c.Block(uint8(lx), int16(surfaceY), uint8(lz), 0)
	if ground != p.grass && ground != p.dirt {
		return false
	}

	var (
		logID, leavesID uint32
		trunkHeight     int
	)

	switch treeType {
	case TreeBirch:
		logID, leavesID = p.birchLog, p.birchLeaves
		trunkHeight = 5 + r.IntN(3)
	case TreeSpruce:
		logID, leavesID = p.spruceLog, p.spruceLeaves
		trunkHeight = 6 + r.IntN(4)
	case TreeAcacia:
		logID, leavesID = p.acaciaLog, p.acaciaLeaves
		trunkHeight = 5 + r.IntN(3)
	case TreeDarkOak:
		logID, leavesID = p.darkOakLog, p.darkOakLeaves
		trunkHeight = 6 + r.IntN(3)
	case TreeJungle:
		logID, leavesID = p.jungleLog, p.jungleLeaves
		trunkHeight = 8 + r.IntN(5)
	case TreeCherry:
		logID, leavesID = p.cherryLog, p.cherryLeaves
		trunkHeight = 5 + r.IntN(3)
	case TreePaleOak:
		logID, leavesID = p.paleOakLog, p.paleOakLeaves
		trunkHeight = 5 + r.IntN(3)
	case TreeMangrove:
		logID, leavesID = p.mangroveLog, p.mangroveLeaves
		trunkHeight = 6 + r.IntN(4)
	case TreePoplar:
		logID, leavesID = p.poplarLog, p.poplarLeaves
		trunkHeight = 6 + r.IntN(3)
	default: // TreeOak
		logID, leavesID = p.oakLog, p.oakLeaves
		trunkHeight = 4 + r.IntN(3)
	}

	// 1. Trunk
	for th := 1; th <= trunkHeight; th++ {
		c.SetBlock(uint8(lx), int16(surfaceY+th), uint8(lz), 0, logID)
	}

	topY := surfaceY + trunkHeight

	// 2. Canopy layers
	switch treeType {
	case TreeSpruce:
		// Conical tiered canopy
		for cy := topY + 1; cy >= topY-trunkHeight+2; cy-- {
			radius := (topY + 1 - cy) % 2
			if cy <= topY-2 {
				radius++
			}
			p.placeLeafDisk(c, lx, cy, lz, radius, leavesID)
		}
	case TreeDarkOak:
		// Dense canopy
		p.placeLeafSquare(c, lx, topY-2, lz, 2, leavesID)
		p.placeLeafSquare(c, lx, topY-1, lz, 3, leavesID)
		p.placeLeafSquare(c, lx, topY, lz, 2, leavesID)
		p.placeLeafCross(c, lx, topY+1, lz, leavesID)
	case TreePoplar:
		// Tall slender columnar canopy
		for cy := topY - 3; cy <= topY+1; cy++ {
			p.placeLeafCross(c, lx, cy, lz, leavesID)
		}
		if topY+2 < 320 {
			c.SetBlock(uint8(lx), int16(topY+2), uint8(lz), 0, leavesID)
		}
	default:
		// Rounded umbrella/cube canopy
		p.placeLeafSquare(c, lx, topY-1, lz, 2, leavesID)
		p.placeLeafSquare(c, lx, topY, lz, 2, leavesID)
		p.placeLeafSquare(c, lx, topY+1, lz, 1, leavesID)
		p.placeLeafCross(c, lx, topY+2, lz, leavesID)
	}

	// Turn dirt beneath trunk
	c.SetBlock(uint8(lx), int16(surfaceY), uint8(lz), 0, p.dirt)
	return true
}

func (p *VegetationPlacer) placeLeafSquare(c *chunk.Chunk, cx, y, cz, radius int, leafID uint32) {
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			// Leave out corners at maximum radius
			if absInt(dx) == radius && absInt(dz) == radius && radius > 1 {
				continue
			}
			x := cx + dx
			z := cz + dz
			if x >= 0 && x < 16 && z >= 0 && z < 16 && y >= -64 && y < 320 {
				if c.Block(uint8(x), int16(y), uint8(z), 0) == p.air {
					c.SetBlock(uint8(x), int16(y), uint8(z), 0, leafID)
				}
			}
		}
	}
}

func (p *VegetationPlacer) placeLeafCross(c *chunk.Chunk, cx, y, cz int, leafID uint32) {
	offsets := [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for _, off := range offsets {
		x := cx + off[0]
		z := cz + off[1]
		if x >= 0 && x < 16 && z >= 0 && z < 16 && y >= -64 && y < 320 {
			if c.Block(uint8(x), int16(y), uint8(z), 0) == p.air {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, leafID)
			}
		}
	}
}

func (p *VegetationPlacer) placeLeafDisk(c *chunk.Chunk, cx, y, cz, radius int, leafID uint32) {
	radSq := radius * radius
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			if dx*dx+dz*dz <= radSq {
				x := cx + dx
				z := cz + dz
				if x >= 0 && x < 16 && z >= 0 && z < 16 && y >= -64 && y < 320 {
					if c.Block(uint8(x), int16(y), uint8(z), 0) == p.air {
						c.SetBlock(uint8(x), int16(y), uint8(z), 0, leafID)
					}
				}
			}
		}
	}
}

// DecorateChunk scatters trees, grass, flowers, cacti, and crops into a chunk column.
func (p *VegetationPlacer) DecorateChunk(c *chunk.Chunk, chunkX, chunkZ int, seed int64) {
	if p == nil || c == nil {
		return
	}
	chunkSeed := uint64(seed) ^ (uint64(chunkX) * 0x3a19b3) ^ (uint64(chunkZ) * 0x712a4f)
	r := rand.New(rand.NewPCG(chunkSeed, chunkSeed^0x1a84f3e5b7c9d012))

	// Find surface heights for each column
	for lx := range 16 {
		for lz := range 16 {
			surfaceY := p.findSurface(c, uint8(lx), uint8(lz))
			if surfaceY <= -50 || surfaceY >= 310 {
				continue
			}

			ground := c.Block(uint8(lx), int16(surfaceY), uint8(lz), 0)
			above := uint16(surfaceY + 1)

			switch ground {
			case p.grass:
				roll := r.Float64()
				switch {
				case roll < 0.25:
					// Short grass
					c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.shortGrass)
				case roll < 0.28:
					// Dandelion
					c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.dandelion)
				case roll < 0.31:
					// Poppy
					c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.poppy)
				case roll < 0.33:
					// Cornflower
					c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.cornflower)
				case roll < 0.35:
					// Mixed tulips & orchids
					flowerRoll := r.IntN(4)
					switch flowerRoll {
					case 0:
						c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.allium)
					case 1:
						c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.redTulip)
					case 2:
						c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.whiteTulip)
					default:
						c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.pinkTulip)
					}
				case roll < 0.365:
					// Double flowers (2-block tall)
					if surfaceY < 310 {
						switch r.IntN(4) {
						case 0:
							c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.sunflowerBottom)
							c.SetBlock(uint8(lx), int16(above+1), uint8(lz), 0, p.sunflowerTop)
						case 1:
							c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.lilacBottom)
							c.SetBlock(uint8(lx), int16(above+1), uint8(lz), 0, p.lilacTop)
						case 2:
							c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.roseBottom)
							c.SetBlock(uint8(lx), int16(above+1), uint8(lz), 0, p.roseTop)
						default:
							c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.peonyBottom)
							c.SetBlock(uint8(lx), int16(above+1), uint8(lz), 0, p.peonyTop)
						}
					}
				case roll < 0.370:
					// Wild Pumpkin
					c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.pumpkin)
				case roll < 0.373:
					// Wild Melon
					c.SetBlock(uint8(lx), int16(above), uint8(lz), 0, p.melon)
				}

			case p.sand:
				// Cacti in desert
				if r.Float64() < 0.03 && lx > 0 && lx < 15 && lz > 0 && lz < 15 {
					height := 1 + r.IntN(3)
					for h := range height {
						c.SetBlock(uint8(lx), int16(above)+int16(h), uint8(lz), 0, p.cactus)
					}
				} else if r.Float64() < 0.05 && p.adjacentToWater(c, uint8(lx), int16(surfaceY), uint8(lz)) {
					// Sugar Cane along shorelines
					height := 1 + r.IntN(3)
					for h := range height {
						c.SetBlock(uint8(lx), int16(above)+int16(h), uint8(lz), 0, p.sugarCane)
					}
				}
			}
		}
	}

	// Scatter trees (1 to 4 trees per chunk)
	treeCount := 1 + r.IntN(4)
	for range treeCount {
		tx := 2 + r.IntN(12)
		tz := 2 + r.IntN(12)
		sy := p.findSurface(c, uint8(tx), uint8(tz))
		if sy > 60 && sy < 250 {
			treeKind := TreeType(r.IntN(10)) // Oak, Birch, Spruce, Acacia, DarkOak, Jungle, Cherry, PaleOak, Mangrove, Poplar
			p.PlaceTree(c, tx, sy, tz, treeKind, r)
		}
	}
}

// adjacentToWater checks if any horizontal neighbor at Y or Y-1 is water.
func (p *VegetationPlacer) adjacentToWater(c *chunk.Chunk, x uint8, y int16, z uint8) bool {
	coords := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for _, crd := range coords {
		nx := int(x) + crd[0]
		nz := int(z) + crd[1]
		if nx >= 0 && nx < 16 && nz >= 0 && nz < 16 {
			b := c.Block(uint8(nx), y, uint8(nz), 0)
			if b == p.water {
				return true
			}
		}
	}
	return false
}

// findSurface locates the highest solid block in column (x, z).
func (p *VegetationPlacer) findSurface(c *chunk.Chunk, x, z uint8) int {
	for y := int16(310); y >= -60; y-- {
		b := c.Block(x, y, z, 0)
		if b != p.air && b != p.water {
			return int(y)
		}
	}
	return -100
}

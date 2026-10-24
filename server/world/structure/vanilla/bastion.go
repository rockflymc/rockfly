package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// BastionRemnant implements world.Structure for the piglin bastion fortress ruins.
type BastionRemnant struct {
	tmpl *structure.Template
}

var _ world.Structure = (*BastionRemnant)(nil)

// NewBastionRemnant constructs a Bastion Remnant fortress with treasure room and gold cache.
func NewBastionRemnant() *BastionRemnant {
	// 26 width (X), 14 height (Y), 26 depth (Z)
	size := cube.Pos{26, 14, 26}
	tmpl := structure.NewTemplate(size)

	blackstone := block.Blackstone{Type: block.NormalBlackstone()}
	gildedBlackstone := block.Blackstone{Type: block.GildedBlackstone()}
	polishedBlackstoneBricks := block.PolishedBlackstoneBrick{}
	goldBlock := block.Gold{}
	lava := block.Lava{Still: true, Depth: 8}
	chest := block.Chest{}

	// 1. Fortress Foundation & Outer Ramparts (Y=0..8)
	for y := range 9 {
		for x := range 26 {
			for z := range 26 {
				isOuter := x == 0 || x == 25 || z == 0 || z == 25
				if isOuter {
					if (x+y+z)%5 == 0 {
						tmpl.SetBlock(cube.Pos{x, y, z}, gildedBlackstone)
					} else {
						tmpl.SetBlock(cube.Pos{x, y, z}, blackstone)
					}
				}
				if y == 0 {
					tmpl.SetBlock(cube.Pos{x, 0, z}, polishedBlackstoneBricks)
				}
			}
		}
	}

	// 2. Central Treasure Chamber (X: 8..17, Z: 8..17, Y: 1..7)
	// Outer surrounding lava moat
	for x := 7; x <= 18; x++ {
		for z := 7; z <= 18; z++ {
			if x == 7 || x == 18 || z == 7 || z == 18 {
				tmpl.SetBlock(cube.Pos{x, 1, z}, lava)
			}
		}
	}

	// Treasure Altar Platform (X: 10..15, Z: 10..15, Y: 1..3)
	for x := 10; x <= 15; x++ {
		for z := 10; z <= 15; z++ {
			tmpl.SetBlock(cube.Pos{x, 1, z}, polishedBlackstoneBricks)
			tmpl.SetBlock(cube.Pos{x, 2, z}, polishedBlackstoneBricks)
		}
	}

	// Gold Block Cache in center of treasure platform (X: 12..13, Z: 12..13, Y: 3..4)
	for x := 12; x <= 13; x++ {
		for z := 12; z <= 13; z++ {
			tmpl.SetBlock(cube.Pos{x, 3, z}, goldBlock)
			tmpl.SetBlock(cube.Pos{x, 4, z}, goldBlock)
		}
	}

	// Double treasure chests on top of altar
	tmpl.SetBlock(cube.Pos{11, 3, 12}, chest)
	tmpl.SetBlock(cube.Pos{14, 3, 13}, chest)

	// 3. Piglin Upper Ramparts (X: 0..25, Z: 0..25, Y: 9)
	for x := range 26 {
		for z := range 26 {
			if (x <= 4 || x >= 21) && (z <= 4 || z >= 21) {
				tmpl.SetBlock(cube.Pos{x, 9, z}, polishedBlackstoneBricks)
				tmpl.SetBlock(cube.Pos{x, 10, z}, blackstone)
			}
		}
	}

	return &BastionRemnant{tmpl: tmpl}
}

// Dimensions returns width, height, and depth.
func (b *BastionRemnant) Dimensions() [3]int {
	d := b.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns the block at local coordinates.
func (b *BastionRemnant) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	blockVal, ok := b.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return blockVal, nil
}

// Build places bastion blocks and registers treasure room bounding box.
func (b *BastionRemnant) Build(tx *world.Tx, origin cube.Pos) *Instance {
	b.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypeBastionRemnant,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(b.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "treasure_room", BBox: NewBoundingBox(origin.Add(cube.Pos{7, 1, 7}), origin.Add(cube.Pos{19, 8, 19}))},
			{Name: "outer_ramparts", BBox: NewBoundingBox(origin, origin.Add(cube.Pos{26, 11, 26}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats bastion blocks directly into a chunk column.
func (b *BastionRemnant) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	b.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

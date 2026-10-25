package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// NetherFortress implements world.Structure for the nether bridge complex.
type NetherFortress struct {
	tmpl *structure.Template
}

var _ world.Structure = (*NetherFortress)(nil)

// NewNetherFortress constructs a Nether Fortress bridge crossing with blaze balcony and nether wart stairs.
func NewNetherFortress() *NetherFortress {
	// 25 width (X), 8 height (Y), 25 depth (Z)
	size := cube.Pos{25, 8, 25}
	tmpl := structure.NewTemplate(size)

	netherBricks := block.NetherBricks{}
	netherFence := block.NetherBrickFence{}
	soulSand := block.SoulSand{}
	netherWart := block.NetherWart{Age: 3}

	// 1. Horizontal Bridge (X: 0..24, Z: 10..14)
	for x := range 25 {
		for z := 10; z <= 14; z++ {
			tmpl.SetBlock(cube.Pos{x, 0, z}, netherBricks)
		}
		// Edge railings
		tmpl.SetBlock(cube.Pos{x, 1, 10}, netherFence)
		tmpl.SetBlock(cube.Pos{x, 1, 14}, netherFence)
	}

	// 2. Vertical Cross-Bridge (Z: 0..24, X: 10..14)
	for z := range 25 {
		for x := 10; x <= 14; x++ {
			tmpl.SetBlock(cube.Pos{x, 0, z}, netherBricks)
		}
		// Edge railings
		tmpl.SetBlock(cube.Pos{10, 1, z}, netherFence)
		tmpl.SetBlock(cube.Pos{14, 1, z}, netherFence)
	}

	// Open the center crossroad (clear fence intersection at 10..14, 10..14)
	for x := 10; x <= 14; x++ {
		for z := 10; z <= 14; z++ {
			tmpl.SetBlock(cube.Pos{x, 1, z}, block.Air{})
		}
	}

	// 3. Blaze Spawner Balcony Platform (X: 17..23, Z: 17..23, Y: 1..5)
	for x := 17; x <= 23; x++ {
		for z := 17; z <= 23; z++ {
			tmpl.SetBlock(cube.Pos{x, 2, z}, netherBricks)
			if x == 17 || x == 23 || z == 17 || z == 23 {
				tmpl.SetBlock(cube.Pos{x, 3, z}, netherFence)
			}
		}
	}
	// Central spawner pedestal
	tmpl.SetBlock(cube.Pos{20, 3, 20}, netherBricks)
	tmpl.SetBlock(cube.Pos{20, 4, 20}, netherFence)

	// 4. Nether Wart Farm Room (X: 1..7, Z: 1..7, Y: 0..3)
	for x := 1; x <= 7; x++ {
		for z := 1; z <= 7; z++ {
			tmpl.SetBlock(cube.Pos{x, 0, z}, netherBricks)
			if x == 1 || x == 7 || z == 1 || z == 7 {
				tmpl.SetBlock(cube.Pos{x, 1, z}, netherBricks)
				tmpl.SetBlock(cube.Pos{x, 2, z}, netherBricks)
			}
		}
	}
	// Soul sand planter rows with mature nether wart
	for x := 3; x <= 5; x++ {
		tmpl.SetBlock(cube.Pos{x, 1, 3}, soulSand)
		tmpl.SetBlock(cube.Pos{x, 2, 3}, netherWart)
		tmpl.SetBlock(cube.Pos{x, 1, 5}, soulSand)
		tmpl.SetBlock(cube.Pos{x, 2, 5}, netherWart)
	}

	return &NetherFortress{tmpl: tmpl}
}

// Dimensions returns width, height, and depth.
func (f *NetherFortress) Dimensions() [3]int {
	d := f.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns the block at local coordinates.
func (f *NetherFortress) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	b, ok := f.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return b, nil
}

// Build places fortress blocks and registers blaze spawner balcony bounding boxes.
func (f *NetherFortress) Build(tx *world.Tx, origin cube.Pos) *Instance {
	f.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypeNetherFortress,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(f.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "bridge_crossing", BBox: NewBoundingBox(origin.Add(cube.Pos{0, 0, 0}), origin.Add(cube.Pos{25, 4, 25}))},
			{Name: "blaze_spawner_balcony", BBox: NewBoundingBox(origin.Add(cube.Pos{17, 1, 17}), origin.Add(cube.Pos{24, 6, 24}))},
			{Name: "nether_wart_room", BBox: NewBoundingBox(origin.Add(cube.Pos{1, 0, 1}), origin.Add(cube.Pos{8, 4, 8}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats nether fortress blocks directly into a chunk column.
func (f *NetherFortress) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	f.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

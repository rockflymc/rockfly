package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// Mineshaft implements world.Structure for abandoned subterranean mineshafts.
type Mineshaft struct {
	tmpl *structure.Template
}

var _ world.Structure = (*Mineshaft)(nil)

// NewMineshaft constructs a 4-way subterranean mineshaft intersection with support arches.
func NewMineshaft() *Mineshaft {
	// 21 width (X), 5 height (Y), 21 depth (Z)
	size := cube.Pos{21, 5, 21}
	tmpl := structure.NewTemplate(size)

	oakPlanks := block.Planks{Wood: block.OakWood()}
	oakFence := block.WoodFence{Wood: block.OakWood()}
	cobweb := block.Cobweb{}
	torch := block.Torch{}

	// Center 4-way crossing: X: 9..11, Z: 9..11
	// Air cavity for corridor across X: 0..20, Z: 9..11 (Y: 1..3)
	// and corridor across Z: 0..20, X: 9..11 (Y: 1..3)
	for y := 1; y <= 3; y++ {
		for x := range 21 {
			for z := 9; z <= 11; z++ {
				tmpl.SetBlock(cube.Pos{x, y, z}, block.Air{})
			}
		}
		for z := range 21 {
			for x := 9; x <= 11; x++ {
				tmpl.SetBlock(cube.Pos{x, y, z}, block.Air{})
			}
		}
	}

	// Floor planks (Y=0)
	for x := range 21 {
		for z := 9; z <= 11; z++ {
			tmpl.SetBlock(cube.Pos{x, 0, z}, oakPlanks)
		}
	}
	for z := range 21 {
		for x := 9; x <= 11; x++ {
			tmpl.SetBlock(cube.Pos{x, 0, z}, oakPlanks)
		}
	}

	// Wooden support arches every 5 blocks along X corridors: at x=4, x=16
	for _, x := range []int{4, 16} {
		tmpl.SetBlock(cube.Pos{x, 1, 9}, oakFence)
		tmpl.SetBlock(cube.Pos{x, 2, 9}, oakFence)
		tmpl.SetBlock(cube.Pos{x, 1, 11}, oakFence)
		tmpl.SetBlock(cube.Pos{x, 2, 11}, oakFence)

		tmpl.SetBlock(cube.Pos{x, 3, 9}, oakPlanks)
		tmpl.SetBlock(cube.Pos{x, 3, 10}, oakPlanks)
		tmpl.SetBlock(cube.Pos{x, 3, 11}, oakPlanks)

		tmpl.SetBlock(cube.Pos{x, 2, 10}, torch)
	}

	// Wooden support arches along Z corridors: at z=4, z=16
	for _, z := range []int{4, 16} {
		tmpl.SetBlock(cube.Pos{9, 1, z}, oakFence)
		tmpl.SetBlock(cube.Pos{9, 2, z}, oakFence)
		tmpl.SetBlock(cube.Pos{11, 1, z}, oakFence)
		tmpl.SetBlock(cube.Pos{11, 2, z}, oakFence)

		tmpl.SetBlock(cube.Pos{9, 3, z}, oakPlanks)
		tmpl.SetBlock(cube.Pos{10, 3, z}, oakPlanks)
		tmpl.SetBlock(cube.Pos{11, 3, z}, oakPlanks)

		tmpl.SetBlock(cube.Pos{10, 2, z}, torch)
	}

	// Cobwebs in upper ceiling corners
	tmpl.SetBlock(cube.Pos{3, 3, 9}, cobweb)
	tmpl.SetBlock(cube.Pos{17, 3, 11}, cobweb)
	tmpl.SetBlock(cube.Pos{9, 3, 3}, cobweb)
	tmpl.SetBlock(cube.Pos{11, 3, 17}, cobweb)

	return &Mineshaft{tmpl: tmpl}
}

// Dimensions returns width, height, and depth.
func (m *Mineshaft) Dimensions() [3]int {
	d := m.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns the block at local coordinates.
func (m *Mineshaft) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	b, ok := m.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return b, nil
}

// Build places mineshaft blocks and registers piece bounding boxes.
func (m *Mineshaft) Build(tx *world.Tx, origin cube.Pos) *Instance {
	m.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypeMineshaft,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(m.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "corridor_x", BBox: NewBoundingBox(origin.Add(cube.Pos{0, 0, 9}), origin.Add(cube.Pos{21, 5, 12}))},
			{Name: "corridor_z", BBox: NewBoundingBox(origin.Add(cube.Pos{9, 0, 0}), origin.Add(cube.Pos{12, 5, 21}))},
			{Name: "crossing", BBox: NewBoundingBox(origin.Add(cube.Pos{9, 0, 9}), origin.Add(cube.Pos{12, 5, 12}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats mineshaft blocks directly into a chunk column.
func (m *Mineshaft) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	m.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

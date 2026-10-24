package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// WitchHut implements world.Structure for the swamp witch hut stilt house.
type WitchHut struct {
	tmpl *structure.Template
}

var _ world.Structure = (*WitchHut)(nil)

// NewWitchHut constructs a 1:1 Swamp Witch Hut.
func NewWitchHut() *WitchHut {
	// 7 width (X), 9 height (Y), 7 depth (Z)
	size := cube.Pos{7, 9, 7}
	tmpl := structure.NewTemplate(size)

	sprucePlanks := block.Planks{Wood: block.SpruceWood()}
	spruceLog := block.Log{Wood: block.SpruceWood()}
	spruceFence := block.WoodFence{Wood: block.SpruceWood()}
	craftingTable := block.CraftingTable{}

	// 1. Stilts (Y=0..2) at corners (1, 1), (5, 1), (1, 5), (5, 5)
	stilts := [][2]int{{1, 1}, {5, 1}, {1, 5}, {5, 5}}
	for _, s := range stilts {
		for y := range 3 {
			tmpl.SetBlock(cube.Pos{s[0], y, s[1]}, spruceLog)
		}
	}

	// 2. Floor at Y=3 (size 5x5 within 1..5)
	for x := 1; x <= 5; x++ {
		for z := 1; z <= 5; z++ {
			tmpl.SetBlock(cube.Pos{x, 3, z}, sprucePlanks)
		}
	}

	// 3. Walls at Y=4..6 (corner logs, plank walls)
	for y := 4; y <= 6; y++ {
		for x := 1; x <= 5; x++ {
			for z := 1; z <= 5; z++ {
				isCorner := (x == 1 || x == 5) && (z == 1 || z == 5)
				isEdge := x == 1 || x == 5 || z == 1 || z == 5
				if isCorner {
					tmpl.SetBlock(cube.Pos{x, y, z}, spruceLog)
				} else if isEdge {
					// Leave door entrance at (3, 4, 1) and (3, 5, 1)
					if x == 3 && z == 1 && (y == 4 || y == 5) {
						continue
					}
					// Leave window at (1, 5, 3)
					if x == 1 && z == 3 && y == 5 {
						continue
					}
					tmpl.SetBlock(cube.Pos{x, y, z}, sprucePlanks)
				}
			}
		}
	}

	// 4. Interior furnishings (Y=4)
	tmpl.SetBlock(cube.Pos{2, 4, 4}, craftingTable)

	// 5. Small front porch fence (Y=3..4)
	tmpl.SetBlock(cube.Pos{3, 3, 0}, sprucePlanks)
	tmpl.SetBlock(cube.Pos{2, 4, 0}, spruceFence)
	tmpl.SetBlock(cube.Pos{4, 4, 0}, spruceFence)

	// 6. Roof (Y=7..8)
	for x := 0; x <= 6; x++ {
		for z := 0; z <= 6; z++ {
			tmpl.SetBlock(cube.Pos{x, 7, z}, sprucePlanks)
		}
	}
	for x := 1; x <= 5; x++ {
		for z := 1; z <= 5; z++ {
			tmpl.SetBlock(cube.Pos{x, 8, z}, sprucePlanks)
		}
	}

	return &WitchHut{tmpl: tmpl}
}

// Dimensions returns width, height, and depth.
func (w *WitchHut) Dimensions() [3]int {
	d := w.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns the block at local coordinates.
func (w *WitchHut) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	b, ok := w.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return b, nil
}

// Build places witch hut blocks and registers witch spawning bounding box.
func (w *WitchHut) Build(tx *world.Tx, origin cube.Pos) *Instance {
	w.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypeWitchHut,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(w.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "witch_spawning_box", BBox: NewBoundingBox(origin.Add(cube.Pos{1, 3, 1}), origin.Add(cube.Pos{6, 7, 6}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats witch hut blocks directly into a chunk column.
func (w *WitchHut) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	w.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

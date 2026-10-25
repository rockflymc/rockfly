package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// PillagerOutpost implements world.Structure for the Pillager watchtower.
type PillagerOutpost struct {
	tmpl *structure.Template
}

var _ world.Structure = (*PillagerOutpost)(nil)

// NewPillagerOutpost constructs a 12x16x12 Pillager Outpost watchtower.
func NewPillagerOutpost() *PillagerOutpost {
	size := cube.Pos{12, 16, 12}
	tmpl := structure.NewTemplate(size)

	cobblestone := block.Cobblestone{}
	darkOakPlanks := block.Planks{Wood: block.DarkOakWood()}
	darkOakLog := block.Log{Wood: block.DarkOakWood()}
	darkOakFence := block.WoodFence{Wood: block.DarkOakWood()}
	chest := block.Chest{}

	// 1. Cobblestone base & corner support pillars (Y=0..4)
	for y := range 5 {
		for x := 1; x <= 10; x++ {
			for z := 1; z <= 10; z++ {
				isOuter := x == 1 || x == 10 || z == 1 || z == 10
				isCornerPillar := (x <= 3 || x >= 8) && (z <= 3 || z >= 8)
				if isCornerPillar || (isOuter && y == 0) {
					tmpl.SetBlock(cube.Pos{x, y, z}, cobblestone)
				}
			}
		}
	}

	// 2. Middle levels (Y=5..11): Dark oak logs at corners, dark oak plank walls
	for y := 5; y <= 11; y++ {
		for x := 2; x <= 9; x++ {
			for z := 2; z <= 9; z++ {
				isCorner := (x == 2 || x == 9) && (z == 2 || z == 9)
				isWall := x == 2 || x == 9 || z == 2 || z == 9
				if isCorner {
					tmpl.SetBlock(cube.Pos{x, y, z}, darkOakLog)
				} else if isWall {
					// Window openings at Y=7 and Y=10
					if (y == 7 || y == 10) && (x == 5 || x == 6 || z == 5 || z == 6) {
						continue
					}
					tmpl.SetBlock(cube.Pos{x, y, z}, darkOakPlanks)
				}
				// Floors at Y=5, Y=8, Y=11
				if (y == 5 || y == 8 || y == 11) && !isWall {
					tmpl.SetBlock(cube.Pos{x, y, z}, darkOakPlanks)
				}
			}
		}
	}

	// 3. Top Balcony & Railings (Y=12)
	for x := 1; x <= 10; x++ {
		for z := 1; z <= 10; z++ {
			// Balcony deck
			tmpl.SetBlock(cube.Pos{x, 12, z}, darkOakPlanks)
			// Outer railings
			if x == 1 || x == 10 || z == 1 || z == 10 {
				tmpl.SetBlock(cube.Pos{x, 13, z}, darkOakFence)
			}
		}
	}

	// 4. Chest Room & Roof canopy (Y=13..15)
	for x := 3; x <= 8; x++ {
		for z := 3; z <= 8; z++ {
			// Top canopy posts
			if (x == 3 || x == 8) && (z == 3 || z == 8) {
				tmpl.SetBlock(cube.Pos{x, 13, z}, darkOakFence)
				tmpl.SetBlock(cube.Pos{x, 14, z}, darkOakFence)
			}
			// Roof canopy cover at Y=15
			tmpl.SetBlock(cube.Pos{x, 15, z}, darkOakPlanks)
		}
	}

	// Loot chest on top floor
	tmpl.SetBlock(cube.Pos{5, 12, 5}, chest)

	return &PillagerOutpost{tmpl: tmpl}
}

// Dimensions returns width, height, and depth.
func (p *PillagerOutpost) Dimensions() [3]int {
	d := p.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns the block at local coordinates.
func (p *PillagerOutpost) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	b, ok := p.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return b, nil
}

// Build places outpost blocks into world tx and registers bounding boxes.
func (p *PillagerOutpost) Build(tx *world.Tx, origin cube.Pos) *Instance {
	p.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypePillagerOutpost,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(p.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "watchtower", BBox: NewBoundingBox(origin, origin.Add(cube.Pos{12, 16, 12}))},
			{Name: "captain_spawning_box", BBox: NewBoundingBox(origin.Add(cube.Pos{1, 12, 1}), origin.Add(cube.Pos{11, 16, 11}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats outpost blocks directly into a chunk column.
func (p *PillagerOutpost) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	p.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

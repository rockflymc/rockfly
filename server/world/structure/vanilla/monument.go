package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// OceanMonument implements world.Structure for the underwater prismarine temple.
type OceanMonument struct {
	tmpl *structure.Template
}

var _ world.Structure = (*OceanMonument)(nil)

// NewOceanMonument constructs an Ocean Monument structure.
func NewOceanMonument() *OceanMonument {
	// 32 width (X), 18 height (Y), 32 depth (Z)
	size := cube.Pos{32, 18, 32}
	tmpl := structure.NewTemplate(size)

	normalPris := block.Prismarine{Type: block.NormalPrismarine()}
	brickPris := block.Prismarine{Type: block.BrickPrismarine()}
	darkPris := block.Prismarine{Type: block.DarkPrismarine()}
	seaLantern := block.SeaLantern{}
	goldBlock := block.Gold{}
	water := block.Water{Still: true, Depth: 8}

	// 1. Monument Base (Y=0..3): 32x32 prismarine perimeter and pillars
	for x := range 32 {
		for z := range 32 {
			tmpl.SetBlock(cube.Pos{x, 0, z}, normalPris)
			if x%4 == 0 || z%4 == 0 {
				tmpl.SetBlock(cube.Pos{x, 1, z}, brickPris)
				tmpl.SetBlock(cube.Pos{x, 2, z}, brickPris)
			}
		}
	}

	// 2. Stepped Temple Tiers (Y=4..12)
	for step := range 5 {
		y := 4 + step*2
		minXZ := step * 3
		maxXZ := 31 - (step * 3)
		for x := minXZ; x <= maxXZ; x++ {
			for z := minXZ; z <= maxXZ; z++ {
				if x == minXZ || x == maxXZ || z == minXZ || z == maxXZ {
					tmpl.SetBlock(cube.Pos{x, y, z}, brickPris)
					tmpl.SetBlock(cube.Pos{x, y + 1, z}, normalPris)
				}
			}
		}
	}

	// 3. Central Treasure Chamber (X: 13..18, Y: 4..9, Z: 13..18)
	// Outer dark prismarine shell
	for x := 13; x <= 18; x++ {
		for y := 4; y <= 9; y++ {
			for z := 13; z <= 18; z++ {
				isOuter := x == 13 || x == 18 || y == 4 || y == 9 || z == 13 || z == 18
				if isOuter {
					tmpl.SetBlock(cube.Pos{x, y, z}, darkPris)
				} else {
					tmpl.SetBlock(cube.Pos{x, y, z}, water)
				}
			}
		}
	}

	// 8 Gold Blocks (2x2x2 cube) in center of treasure room (X: 15..16, Y: 6..7, Z: 15..16)
	for x := 15; x <= 16; x++ {
		for y := 6; y <= 7; y++ {
			for z := 15; z <= 16; z++ {
				tmpl.SetBlock(cube.Pos{x, y, z}, goldBlock)
			}
		}
	}

	// 4. Sea Lantern Beacons on exterior corners
	cornerLanterns := [][2]int{{3, 3}, {28, 3}, {3, 28}, {28, 28}, {15, 15}}
	for _, cl := range cornerLanterns {
		tmpl.SetBlock(cube.Pos{cl[0], 5, cl[1]}, seaLantern)
		tmpl.SetBlock(cube.Pos{cl[0], 11, cl[1]}, seaLantern)
	}

	// 5. Top Elder Guardian Chamber (X: 12..19, Y: 13..17, Z: 12..19)
	for x := 12; x <= 19; x++ {
		for z := 12; z <= 19; z++ {
			tmpl.SetBlock(cube.Pos{x, 13, z}, darkPris)
			tmpl.SetBlock(cube.Pos{x, 17, z}, brickPris)
			if x == 12 || x == 19 || z == 12 || z == 19 {
				tmpl.SetBlock(cube.Pos{x, 14, z}, normalPris)
				tmpl.SetBlock(cube.Pos{x, 15, z}, normalPris)
				tmpl.SetBlock(cube.Pos{x, 16, z}, normalPris)
			}
		}
	}
	tmpl.SetBlock(cube.Pos{15, 14, 15}, seaLantern)

	return &OceanMonument{tmpl: tmpl}
}

// Dimensions returns width, height, and depth.
func (m *OceanMonument) Dimensions() [3]int {
	d := m.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns the block at local coordinates.
func (m *OceanMonument) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	b, ok := m.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return b, nil
}

// Build places ocean monument blocks and registers guardian spawning bounding boxes.
func (m *OceanMonument) Build(tx *world.Tx, origin cube.Pos) *Instance {
	m.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypeOceanMonument,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(m.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "temple_base", BBox: NewBoundingBox(origin, origin.Add(cube.Pos{32, 13, 32}))},
			{Name: "treasure_chamber", BBox: NewBoundingBox(origin.Add(cube.Pos{13, 4, 13}), origin.Add(cube.Pos{19, 10, 19}))},
			{Name: "elder_guardian_top", BBox: NewBoundingBox(origin.Add(cube.Pos{12, 13, 12}), origin.Add(cube.Pos{20, 18, 20}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats ocean monument blocks directly into a chunk column.
func (m *OceanMonument) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	m.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

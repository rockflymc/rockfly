package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// DesertPyramid implements world.Structure for the canonical 21x13x21 desert temple.
type DesertPyramid struct {
	tmpl *structure.Template
}

var _ world.Structure = (*DesertPyramid)(nil)

// NewDesertPyramid constructs a full 1:1 Desert Pyramid structure template.
func NewDesertPyramid() *DesertPyramid {
	// Size: 21 width (X), 19 total height (Y) including subterranean secret chamber, 21 depth (Z)
	size := cube.Pos{21, 19, 21}
	tmpl := structure.NewTemplate(size)

	sandstone := block.Sandstone{Type: block.NormalSandstone()}
	smoothSandstone := block.Sandstone{Type: block.SmoothSandstone()}
	cutSandstone := block.Sandstone{Type: block.CutSandstone()}
	orangeTerracotta := block.StainedTerracotta{Colour: item.ColourOrange()}
	blueTerracotta := block.StainedTerracotta{Colour: item.ColourBlue()}
	tnt := block.TNT{}

	// Ground level starts at Y=6. Y=0..5 is secret subterranean chamber.
	// Step 1: Subterranean chamber (X: 8..12, Z: 8..12, Y: 0..5)
	for x := 8; x <= 12; x++ {
		for z := 8; z <= 12; z++ {
			// Basement floor at Y=0
			tmpl.SetBlock(cube.Pos{x, 0, z}, cutSandstone)
			// TNT layer at Y=1
			tmpl.SetBlock(cube.Pos{x, 1, z}, tnt)
			// Sandstone cover over TNT at Y=2
			tmpl.SetBlock(cube.Pos{x, 2, z}, smoothSandstone)

			// Air cavity Y=3..5
			for y := 3; y <= 5; y++ {
				if x == 8 || x == 12 || z == 8 || z == 12 {
					// Basement walls
					tmpl.SetBlock(cube.Pos{x, y, z}, sandstone)
				}
			}
		}
	}

	// Step 2: Main Pyramid Base (Y=6) and stepped tiers (Y=6..14)
	for step := range 9 {
		y := 6 + step
		minXZ := step
		maxXZ := 20 - step
		for x := minXZ; x <= maxXZ; x++ {
			for z := minXZ; z <= maxXZ; z++ {
				// Outer shell
				if x == minXZ || x == maxXZ || z == minXZ || z == maxXZ {
					tmpl.SetBlock(cube.Pos{x, y, z}, sandstone)
				} else if y == 6 {
					// Solid ground floor
					tmpl.SetBlock(cube.Pos{x, y, z}, smoothSandstone)
				}
			}
		}
	}

	// Step 3: Decorative Centerpiece Floor (Y=6)
	// Blue terracotta center cross with orange accents
	tmpl.SetBlock(cube.Pos{10, 6, 10}, blueTerracotta)
	tmpl.SetBlock(cube.Pos{9, 6, 10}, orangeTerracotta)
	tmpl.SetBlock(cube.Pos{11, 6, 10}, orangeTerracotta)
	tmpl.SetBlock(cube.Pos{10, 6, 9}, orangeTerracotta)
	tmpl.SetBlock(cube.Pos{10, 6, 11}, orangeTerracotta)

	// Step 4: Two Front Corner Towers (X: 0..4, Z: 0..4) and (X: 16..20, Z: 0..4)
	buildPyramidTower(tmpl, 0, 0, sandstone, orangeTerracotta, cutSandstone)
	buildPyramidTower(tmpl, 16, 0, sandstone, orangeTerracotta, cutSandstone)

	return &DesertPyramid{tmpl: tmpl}
}

func buildPyramidTower(tmpl *structure.Template, startX, startZ int, sand, orange, cut world.Block) {
	for y := 6; y <= 16; y++ {
		for x := startX; x <= startX+4; x++ {
			for z := startZ; z <= startZ+4; z++ {
				if x == startX || x == startX+4 || z == startZ || z == startZ+4 {
					// Corner tower walls
					if y == 10 || y == 14 {
						tmpl.SetBlock(cube.Pos{x, y, z}, orange)
					} else {
						tmpl.SetBlock(cube.Pos{x, y, z}, sand)
					}
				}
			}
		}
	}
	// Tower cap
	for x := startX + 1; x <= startX+3; x++ {
		for z := startZ + 1; z <= startZ+3; z++ {
			tmpl.SetBlock(cube.Pos{x, 17, z}, cut)
		}
	}
}

// Dimensions returns width, height, and depth dimensions.
func (p *DesertPyramid) Dimensions() [3]int {
	d := p.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns block at local coordinates.
func (p *DesertPyramid) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	b, ok := p.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return b, nil
}

// Build places pyramid blocks into the world transaction.
func (p *DesertPyramid) Build(tx *world.Tx, origin cube.Pos) *Instance {
	p.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypeDesertPyramid,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(p.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "pyramid_base", BBox: NewBoundingBox(origin.Add(cube.Pos{0, 6, 0}), origin.Add(cube.Pos{21, 15, 21}))},
			{Name: "secret_chamber", BBox: NewBoundingBox(origin.Add(cube.Pos{8, 0, 8}), origin.Add(cube.Pos{12, 6, 12}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats pyramid blocks directly into a chunk column.
func (p *DesertPyramid) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	p.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

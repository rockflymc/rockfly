package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// VillageTheme defines the architectural style of a village.
type VillageTheme int

const (
	VillagePlains VillageTheme = iota
	VillageDesert
	VillageTaiga
	VillageSavanna
	VillageSnowy
)

// Village implements world.Structure for multi-building vanilla villages.
type Village struct {
	theme VillageTheme
	tmpl  *structure.Template
}

var _ world.Structure = (*Village)(nil)

// NewVillage constructs a multi-piece village complex with a town well, houses, blacksmith, farm, and connecting paths.
func NewVillage(theme VillageTheme) *Village {
	// Size: 32 width (X), 12 height (Y), 32 depth (Z)
	size := cube.Pos{32, 12, 32}
	tmpl := structure.NewTemplate(size)

	// Material selection by theme
	var (
		plankBlock     world.Block = block.Planks{Wood: block.OakWood()}
		logBlock       world.Block = block.Log{Wood: block.OakWood()}
		fenceBlock     world.Block = block.WoodFence{Wood: block.OakWood()}
		foundation     world.Block = block.Cobblestone{}
		wallBlock      world.Block = block.Cobblestone{}
		pathBlock      world.Block = block.DirtPath{}
		waterBlock     world.Block = block.Water{Still: true, Depth: 8}
		lavaBlock      world.Block = block.Lava{Still: true, Depth: 8}
		farmBlock      world.Block = block.Farmland{}
		furnaceBlock   world.Block = block.Furnace{}
		ironBarsBlock  world.Block = block.IronBars{}
		composterBlock world.Block = block.Composter{}
		glassBlock     world.Block = block.Glass{}
	)

	switch theme {
	case VillageDesert:
		plankBlock = block.Sandstone{Type: block.SmoothSandstone()}
		logBlock = block.Sandstone{Type: block.CutSandstone()}
		foundation = block.Sandstone{Type: block.NormalSandstone()}
		wallBlock = block.Sandstone{Type: block.NormalSandstone()}
	case VillageTaiga:
		plankBlock = block.Planks{Wood: block.SpruceWood()}
		logBlock = block.Log{Wood: block.SpruceWood()}
		fenceBlock = block.WoodFence{Wood: block.SpruceWood()}
	case VillageSavanna:
		plankBlock = block.Planks{Wood: block.AcaciaWood()}
		logBlock = block.Log{Wood: block.AcaciaWood()}
		fenceBlock = block.WoodFence{Wood: block.AcaciaWood()}
	}

	// 1. Town Center Well (X: 13..18, Z: 13..18)
	// Base rim (Y=0), water basin in center (14..17, 14..17)
	for x := 13; x <= 18; x++ {
		for z := 13; z <= 18; z++ {
			if x == 13 || x == 18 || z == 13 || z == 18 {
				tmpl.SetBlock(cube.Pos{x, 0, z}, wallBlock)
				tmpl.SetBlock(cube.Pos{x, 1, z}, wallBlock)
			} else {
				tmpl.SetBlock(cube.Pos{x, 0, z}, waterBlock)
			}
		}
	}
	// Well corner posts (Y=2..4) and roof (Y=5)
	wellCorners := [][2]int{{13, 13}, {18, 13}, {13, 18}, {18, 18}}
	for _, c := range wellCorners {
		for y := 2; y <= 4; y++ {
			tmpl.SetBlock(cube.Pos{c[0], y, c[1]}, fenceBlock)
		}
	}
	for x := 13; x <= 18; x++ {
		for z := 13; z <= 18; z++ {
			tmpl.SetBlock(cube.Pos{x, 5, z}, plankBlock)
		}
	}

	// 2. North Residential House (X: 13..18, Z: 2..7)
	buildVillageHouse(tmpl, 13, 2, foundation, wallBlock, plankBlock, logBlock, glassBlock)

	// 3. West Small House (X: 2..7, Z: 13..18)
	buildVillageHouse(tmpl, 2, 13, foundation, wallBlock, plankBlock, logBlock, glassBlock)

	// 4. East Blacksmith / Weaponsmith (X: 22..29, Z: 13..19)
	buildVillageBlacksmith(tmpl, 22, 13, foundation, wallBlock, plankBlock, lavaBlock, furnaceBlock, ironBarsBlock)

	// 5. South Crop Farm (X: 11..20, Z: 22..30)
	buildVillageFarm(tmpl, 11, 22, logBlock, farmBlock, waterBlock, composterBlock)

	// 6. Connecting Street Network (3-wide dirt paths)
	// Vertical spine: X: 14..16, Z: 0..31
	for z := range 32 {
		for x := 14; x <= 16; x++ {
			if tmplBlock, ok := tmpl.Block(cube.Pos{x, 0, z}); !ok || tmplBlock == nil {
				tmpl.SetBlock(cube.Pos{x, 0, z}, pathBlock)
			}
		}
	}
	// Horizontal cross-street: Z: 14..16, X: 0..31
	for x := range 32 {
		for z := 14; z <= 16; z++ {
			if tmplBlock, ok := tmpl.Block(cube.Pos{x, 0, z}); !ok || tmplBlock == nil {
				tmpl.SetBlock(cube.Pos{x, 0, z}, pathBlock)
			}
		}
	}

	return &Village{theme: theme, tmpl: tmpl}
}

func buildVillageHouse(tmpl *structure.Template, startX, startZ int, found, wall, plank, log, glass world.Block) {
	// Floor Y=0, Walls Y=1..3, Roof Y=4..5
	for x := startX; x <= startX+5; x++ {
		for z := startZ; z <= startZ+5; z++ {
			tmpl.SetBlock(cube.Pos{x, 0, z}, found)

			isCorner := (x == startX || x == startX+5) && (z == startZ || z == startZ+5)
			isEdge := x == startX || x == startX+5 || z == startZ || z == startZ+5

			for y := 1; y <= 3; y++ {
				if isCorner {
					tmpl.SetBlock(cube.Pos{x, y, z}, log)
				} else if isEdge {
					// Door opening on south side
					if z == startZ+5 && (x == startX+2) && (y == 1 || y == 2) {
						continue
					}
					// Window on east side
					if x == startX+5 && z == startZ+2 && y == 2 {
						tmpl.SetBlock(cube.Pos{x, y, z}, glass)
					} else {
						tmpl.SetBlock(cube.Pos{x, y, z}, plank)
					}
				}
			}
			// Roof
			tmpl.SetBlock(cube.Pos{x, 4, z}, wall)
		}
	}
}

func buildVillageBlacksmith(tmpl *structure.Template, startX, startZ int, found, wall, plank, lava, furnace, ironBars world.Block) {
	for x := startX; x <= startX+7; x++ {
		for z := startZ; z <= startZ+6; z++ {
			tmpl.SetBlock(cube.Pos{x, 0, z}, found)

			isEdge := x == startX || x == startX+7 || z == startZ || z == startZ+6
			for y := 1; y <= 3; y++ {
				if isEdge {
					tmpl.SetBlock(cube.Pos{x, y, z}, wall)
				}
			}
			tmpl.SetBlock(cube.Pos{x, 4, z}, plank)
		}
	}
	// Lava basin (1x2) and dual furnaces
	tmpl.SetBlock(cube.Pos{startX + 1, 1, startZ + 1}, lava)
	tmpl.SetBlock(cube.Pos{startX + 2, 1, startZ + 1}, lava)
	tmpl.SetBlock(cube.Pos{startX + 1, 1, startZ + 2}, ironBars)
	tmpl.SetBlock(cube.Pos{startX + 2, 1, startZ + 2}, ironBars)
	tmpl.SetBlock(cube.Pos{startX + 4, 1, startZ + 1}, furnace)
	tmpl.SetBlock(cube.Pos{startX + 5, 1, startZ + 1}, furnace)
}

func buildVillageFarm(tmpl *structure.Template, startX, startZ int, log, farm, water, composter world.Block) {
	for x := startX; x <= startX+9; x++ {
		for z := startZ; z <= startZ+8; z++ {
			isBorder := x == startX || x == startX+9 || z == startZ || z == startZ+8
			if isBorder {
				tmpl.SetBlock(cube.Pos{x, 0, z}, log)
			} else if z == startZ+4 {
				// Central water channel
				tmpl.SetBlock(cube.Pos{x, 0, z}, water)
			} else {
				tmpl.SetBlock(cube.Pos{x, 0, z}, farm)
			}
		}
	}
	tmpl.SetBlock(cube.Pos{startX + 1, 1, startZ + 1}, composter)
}

// Dimensions returns width, height, and depth.
func (v *Village) Dimensions() [3]int {
	d := v.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns the block at local coordinates.
func (v *Village) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	b, ok := v.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return b, nil
}

// Build places the village and registers all building bounding boxes.
func (v *Village) Build(tx *world.Tx, origin cube.Pos) *Instance {
	v.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypeVillage,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(v.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "town_well", BBox: NewBoundingBox(origin.Add(cube.Pos{13, 0, 13}), origin.Add(cube.Pos{19, 6, 19}))},
			{Name: "house_north", BBox: NewBoundingBox(origin.Add(cube.Pos{13, 0, 2}), origin.Add(cube.Pos{19, 6, 8}))},
			{Name: "house_west", BBox: NewBoundingBox(origin.Add(cube.Pos{2, 0, 13}), origin.Add(cube.Pos{8, 6, 19}))},
			{Name: "blacksmith", BBox: NewBoundingBox(origin.Add(cube.Pos{22, 0, 13}), origin.Add(cube.Pos{30, 5, 20}))},
			{Name: "farm", BBox: NewBoundingBox(origin.Add(cube.Pos{11, 0, 22}), origin.Add(cube.Pos{21, 2, 31}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats village blocks directly into a chunk column.
func (v *Village) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	v.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

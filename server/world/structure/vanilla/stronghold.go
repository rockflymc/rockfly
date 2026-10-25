package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// Stronghold implements world.Structure for the underground stronghold and portal room.
type Stronghold struct {
	tmpl *structure.Template
}

var _ world.Structure = (*Stronghold)(nil)

// NewStronghold constructs a subterranean Stronghold complex containing a library and the End Portal Room.
func NewStronghold() *Stronghold {
	// 24 width (X), 9 height (Y), 24 depth (Z)
	size := cube.Pos{24, 9, 24}
	tmpl := structure.NewTemplate(size)

	stoneBricks := block.StoneBricks{}
	oakPlanks := block.Planks{Wood: block.OakWood()}
	bookshelf := block.Bookshelf{}
	ironBars := block.IronBars{}
	lava := block.Lava{Still: true, Depth: 8}
	torch := block.Torch{}

	// 1. Central Connecting Corridor (X: 0..23, Y: 0..4, Z: 10..13)
	for x := range 24 {
		for z := 10; z <= 13; z++ {
			tmpl.SetBlock(cube.Pos{x, 0, z}, stoneBricks)
			tmpl.SetBlock(cube.Pos{x, 4, z}, stoneBricks)
			if z == 10 || z == 13 {
				tmpl.SetBlock(cube.Pos{x, 1, z}, stoneBricks)
				tmpl.SetBlock(cube.Pos{x, 2, z}, stoneBricks)
				tmpl.SetBlock(cube.Pos{x, 3, z}, stoneBricks)
			}
		}
	}
	tmpl.SetBlock(cube.Pos{6, 2, 10}, torch)
	tmpl.SetBlock(cube.Pos{18, 2, 13}, torch)

	// 2. Stronghold Library (X: 0..10, Y: 0..8, Z: 0..10)
	for x := range 11 {
		for z := range 11 {
			tmpl.SetBlock(cube.Pos{x, 0, z}, oakPlanks)
			tmpl.SetBlock(cube.Pos{x, 8, z}, oakPlanks)

			isWall := x == 0 || x == 10 || z == 0 || z == 10
			if isWall {
				for y := 1; y <= 7; y++ {
					// Leave doorway into corridor at (5, 1..3, 10)
					if z == 10 && (x == 4 || x == 5 || x == 6) && y <= 3 {
						continue
					}
					if y == 2 || y == 3 || y == 5 || y == 6 {
						tmpl.SetBlock(cube.Pos{x, y, z}, bookshelf)
					} else {
						tmpl.SetBlock(cube.Pos{x, y, z}, stoneBricks)
					}
				}
			}
		}
	}

	// 3. End Portal Room (X: 13..23, Y: 0..7, Z: 13..23)
	// Outer room shell
	for x := 13; x <= 23; x++ {
		for z := 13; z <= 23; z++ {
			tmpl.SetBlock(cube.Pos{x, 0, z}, stoneBricks)
			tmpl.SetBlock(cube.Pos{x, 7, z}, stoneBricks)

			isWall := x == 13 || x == 23 || z == 13 || z == 23
			if isWall {
				for y := 1; y <= 6; y++ {
					// Leave doorway into corridor at (13, 1..3, 11..12)
					if x == 13 && (z == 13 || z == 14) && y <= 3 {
						continue
					}
					tmpl.SetBlock(cube.Pos{x, y, z}, stoneBricks)
				}
			}
		}
	}

	// Portal Room central staircase & raised altar platform
	for x := 16; x <= 20; x++ {
		for z := 16; z <= 20; z++ {
			tmpl.SetBlock(cube.Pos{x, 1, z}, stoneBricks)
		}
	}

	// Lava pool (3x3) inside altar platform (X: 17..19, Z: 17..19) at Y=1
	for x := 17; x <= 19; x++ {
		for z := 17; z <= 19; z++ {
			tmpl.SetBlock(cube.Pos{x, 1, z}, lava)
		}
	}

	// 12 End Portal Frames surrounding lava pool at Y=2
	// North side (Z=16, X: 17..19) facing South
	tmpl.SetBlock(cube.Pos{17, 2, 16}, block.EndPortalFrame{Facing: cube.South, Eye: true})
	tmpl.SetBlock(cube.Pos{18, 2, 16}, block.EndPortalFrame{Facing: cube.South, Eye: false})
	tmpl.SetBlock(cube.Pos{19, 2, 16}, block.EndPortalFrame{Facing: cube.South, Eye: true})

	// South side (Z=20, X: 17..19) facing North
	tmpl.SetBlock(cube.Pos{17, 2, 20}, block.EndPortalFrame{Facing: cube.North, Eye: false})
	tmpl.SetBlock(cube.Pos{18, 2, 20}, block.EndPortalFrame{Facing: cube.North, Eye: true})
	tmpl.SetBlock(cube.Pos{19, 2, 20}, block.EndPortalFrame{Facing: cube.North, Eye: false})

	// West side (X=16, Z: 17..19) facing East
	tmpl.SetBlock(cube.Pos{16, 2, 17}, block.EndPortalFrame{Facing: cube.East, Eye: true})
	tmpl.SetBlock(cube.Pos{16, 2, 18}, block.EndPortalFrame{Facing: cube.East, Eye: false})
	tmpl.SetBlock(cube.Pos{16, 2, 19}, block.EndPortalFrame{Facing: cube.East, Eye: false})

	// East side (X=20, Z: 17..19) facing West
	tmpl.SetBlock(cube.Pos{20, 2, 17}, block.EndPortalFrame{Facing: cube.West, Eye: false})
	tmpl.SetBlock(cube.Pos{20, 2, 18}, block.EndPortalFrame{Facing: cube.West, Eye: true})
	tmpl.SetBlock(cube.Pos{20, 2, 19}, block.EndPortalFrame{Facing: cube.West, Eye: false})

	// Iron bars guarding portal basin
	tmpl.SetBlock(cube.Pos{15, 2, 15}, ironBars)
	tmpl.SetBlock(cube.Pos{21, 2, 15}, ironBars)
	tmpl.SetBlock(cube.Pos{15, 2, 21}, ironBars)
	tmpl.SetBlock(cube.Pos{21, 2, 21}, ironBars)

	return &Stronghold{tmpl: tmpl}
}

// Dimensions returns width, height, and depth.
func (s *Stronghold) Dimensions() [3]int {
	d := s.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns the block at local coordinates.
func (s *Stronghold) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	b, ok := s.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return b, nil
}

// Build places stronghold blocks and registers portal room and library bounding boxes.
func (s *Stronghold) Build(tx *world.Tx, origin cube.Pos) *Instance {
	s.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypeStronghold,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(s.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "portal_room", BBox: NewBoundingBox(origin.Add(cube.Pos{13, 0, 13}), origin.Add(cube.Pos{24, 8, 24}))},
			{Name: "library", BBox: NewBoundingBox(origin.Add(cube.Pos{0, 0, 0}), origin.Add(cube.Pos{11, 9, 11}))},
			{Name: "corridor", BBox: NewBoundingBox(origin.Add(cube.Pos{0, 0, 10}), origin.Add(cube.Pos{24, 5, 14}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats stronghold blocks directly into a chunk column.
func (s *Stronghold) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	s.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

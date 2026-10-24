package vanilla

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// AncientCity implements world.Structure for the subterranean deep dark ancient city.
type AncientCity struct {
	tmpl *structure.Template
}

var _ world.Structure = (*AncientCity)(nil)

// NewAncientCity constructs an Ancient City structure with central portal shrine and ruins.
func NewAncientCity() *AncientCity {
	// 28 width (X), 14 height (Y), 28 depth (Z)
	size := cube.Pos{28, 14, 28}
	tmpl := structure.NewTemplate(size)

	deepslateBricks := block.DeepslateBricks{}
	reinforcedDeepslate := block.ReinforcedDeepslate{}
	basalt := block.Basalt{}
	soulLantern := block.Lantern{Type: block.SoulFire()}

	// 1. Ruined Floor & Corridors (Y=0)
	for x := range 28 {
		for z := range 28 {
			if (x+z)%2 == 0 {
				tmpl.SetBlock(cube.Pos{x, 0, z}, deepslateBricks)
			}
		}
	}

	// 2. Basalt Columns on walkways at corners and path borders
	pedestals := [][2]int{{4, 4}, {23, 4}, {4, 23}, {23, 23}, {9, 10}, {18, 10}}
	for _, p := range pedestals {
		tmpl.SetBlock(cube.Pos{p[0], 1, p[1]}, basalt)
		tmpl.SetBlock(cube.Pos{p[0], 2, p[1]}, basalt)
		tmpl.SetBlock(cube.Pos{p[0], 3, p[1]}, soulLantern)
	}

	// 3. Central Reinforced Deepslate Portal Shrine (X: 8..19, Y: 1..10, Z: 14)
	// Outer vertical frame: 12 blocks wide (X: 8..19), 10 blocks tall (Y: 1..10)
	for x := 8; x <= 19; x++ {
		for y := 1; y <= 10; y++ {
			isFrameEdge := x == 8 || x == 19 || y == 1 || y == 10
			if isFrameEdge {
				tmpl.SetBlock(cube.Pos{x, y, 14}, reinforcedDeepslate)
			}
		}
	}

	// Shrine Base Platform (X: 6..21, Y: 1, Z: 11..17)
	for x := 6; x <= 21; x++ {
		for z := 11; z <= 17; z++ {
			tmpl.SetBlock(cube.Pos{x, 1, z}, deepslateBricks)
		}
	}

	// 4. Ruined Corridor Walls (X: 0..4 and X: 23..27)
	for y := 1; y <= 4; y++ {
		for x := range 4 {
			for z := range 28 {
				if z%4 == 0 {
					tmpl.SetBlock(cube.Pos{x, y, z}, deepslateBricks)
					tmpl.SetBlock(cube.Pos{27 - x, y, z}, deepslateBricks)
				}
			}
		}
	}

	return &AncientCity{tmpl: tmpl}
}

// Dimensions returns width, height, and depth.
func (a *AncientCity) Dimensions() [3]int {
	d := a.tmpl.Dimensions()
	return [3]int{d.X(), d.Y(), d.Z()}
}

// At returns the block at local coordinates.
func (a *AncientCity) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	b, ok := a.tmpl.Block(cube.Pos{x, y, z})
	if !ok {
		return nil, nil
	}
	return b, nil
}

// Build places ancient city blocks and registers warden shrine bounding box.
func (a *AncientCity) Build(tx *world.Tx, origin cube.Pos) *Instance {
	a.tmpl.Build(tx, origin, structure.Rotation0, structure.MirrorNone)

	inst := &Instance{
		Type:   TypeAncientCity,
		Origin: origin,
		BBox:   NewBoundingBox(origin, origin.Add(a.tmpl.Dimensions())),
		Pieces: []Piece{
			{Name: "city_shrine", BBox: NewBoundingBox(origin.Add(cube.Pos{6, 1, 11}), origin.Add(cube.Pos{22, 11, 18}))},
			{Name: "ruins_corridor_west", BBox: NewBoundingBox(origin, origin.Add(cube.Pos{6, 5, 28}))},
			{Name: "ruins_corridor_east", BBox: NewBoundingBox(origin.Add(cube.Pos{22, 0, 0}), origin.Add(cube.Pos{28, 5, 28}))},
		},
	}
	DefaultManager().Add(inst)
	return inst
}

// BuildChunk splats ancient city blocks directly into a chunk column.
func (a *AncientCity) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, reg world.BlockRegistry) {
	a.tmpl.BuildChunk(c, chunkPos, origin, structure.Rotation0, structure.MirrorNone, reg)
}

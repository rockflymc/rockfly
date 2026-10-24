package jigsaw_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
	"github.com/df-mc/dragonfly/server/world/structure/jigsaw"
)

func TestBoundingBoxIntersection(t *testing.T) {
	b1 := jigsaw.NewBoundingBox(cube.Pos{0, 0, 0}, cube.Pos{10, 10, 10})
	b2 := jigsaw.NewBoundingBox(cube.Pos{5, 5, 5}, cube.Pos{10, 10, 10})
	b3 := jigsaw.NewBoundingBox(cube.Pos{20, 20, 20}, cube.Pos{5, 5, 5})

	if !b1.Intersects(b2) {
		t.Fatal("expected b1 and b2 to intersect")
	}
	if b1.Intersects(b3) {
		t.Fatal("expected b1 and b3 not to intersect")
	}
}

func TestRotateFace(t *testing.T) {
	if f := jigsaw.RotateFace(cube.FaceNorth, structure.Rotation90); f != cube.FaceEast {
		t.Fatalf("expected North rotated 90 to be East, got %v", f)
	}
	if f := jigsaw.RotateFace(cube.FaceNorth, structure.Rotation180); f != cube.FaceSouth {
		t.Fatalf("expected North rotated 180 to be South, got %v", f)
	}
	if f := jigsaw.RotateFace(cube.FaceNorth, structure.Rotation270); f != cube.FaceWest {
		t.Fatalf("expected North rotated 270 to be West, got %v", f)
	}
	if f := jigsaw.RotateFace(cube.FaceUp, structure.Rotation90); f != cube.FaceUp {
		t.Fatalf("expected Up face to remain Up, got %v", f)
	}
}

func TestJigsawAssembly(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	stone := block.Stone{}
	dirt := block.Dirt{}

	// Piece 1: Starter Room (3x3x3) with East-facing exit at (2, 1, 1)
	t1 := structure.NewTemplate(cube.Pos{3, 3, 3})
	t1.SetBlock(cube.Pos{0, 0, 0}, stone)
	piece1 := &jigsaw.Piece{
		Template: t1,
		Jigsaws: []jigsaw.JigsawBlock{
			{
				Pos:        cube.Pos{2, 1, 1},
				Name:       "starter_exit",
				Target:     "corridors",
				Facing:     cube.FaceEast,
				FinalBlock: dirt,
			},
		},
	}

	// Piece 2: Corridor (3x3x3) with West-facing input at (0, 1, 1) and East-facing output at (2, 1, 1)
	t2 := structure.NewTemplate(cube.Pos{3, 3, 3})
	t2.SetBlock(cube.Pos{1, 1, 1}, stone)
	piece2 := &jigsaw.Piece{
		Template: t2,
		Jigsaws: []jigsaw.JigsawBlock{
			{
				Pos:        cube.Pos{0, 1, 1},
				Name:       "corridor_in",
				Target:     "corridors",
				Facing:     cube.FaceWest,
				FinalBlock: dirt,
			},
			{
				Pos:        cube.Pos{2, 1, 1},
				Name:       "corridor_out",
				Target:     "corridors",
				Facing:     cube.FaceEast,
				FinalBlock: dirt,
			},
		},
	}

	assembler := jigsaw.NewAssembler()
	assembler.RegisterPool(jigsaw.NewPool("start", "", jigsaw.WeightedPiece{Piece: piece1, Weight: 1}))
	assembler.RegisterPool(jigsaw.NewPool("corridors", "", jigsaw.WeightedPiece{Piece: piece2, Weight: 1}))

	placed := assembler.Assemble("start", cube.Pos{0, 64, 0}, 3, 12345)

	if len(placed) < 2 {
		t.Fatalf("expected at least 2 pieces assembled, got %d", len(placed))
	}

	// Verify pieces do not collide
	for i := 0; i < len(placed); i++ {
		for j := i + 1; j < len(placed); j++ {
			if placed[i].Box.Intersects(placed[j].Box) {
				t.Fatalf("piece %d collides with piece %d: %v vs %v", i, j, placed[i].Box, placed[j].Box)
			}
		}
	}

	// Verify chunk population
	c := chunk.New(reg, cube.Range{0, 255})
	assembler.BuildChunk(c, world.ChunkPos{0, 0}, placed, reg)

	dirtRID := reg.BlockRuntimeID(dirt)
	// At connection point (2, 65, 1) finalBlock dirt should be placed
	if b := c.Block(2, 65, 1, 0); b != dirtRID {
		t.Fatalf("expected final block dirt at (2, 65, 1), got %d", b)
	}
}

func BenchmarkJigsawAssembly(b *testing.B) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	t1 := structure.NewTemplate(cube.Pos{3, 3, 3})
	p1 := &jigsaw.Piece{
		Template: t1,
		Jigsaws: []jigsaw.JigsawBlock{
			{Pos: cube.Pos{2, 1, 1}, Name: "exit", Target: "pool", Facing: cube.FaceEast},
		},
	}

	t2 := structure.NewTemplate(cube.Pos{3, 3, 3})
	p2 := &jigsaw.Piece{
		Template: t2,
		Jigsaws: []jigsaw.JigsawBlock{
			{Pos: cube.Pos{0, 1, 1}, Name: "in", Target: "pool", Facing: cube.FaceWest},
		},
	}

	assembler := jigsaw.NewAssembler()
	assembler.RegisterPool(jigsaw.NewPool("start", "", jigsaw.WeightedPiece{Piece: p1, Weight: 1}))
	assembler.RegisterPool(jigsaw.NewPool("pool", "", jigsaw.WeightedPiece{Piece: p2, Weight: 1}))

	c := chunk.New(reg, cube.Range{0, 255})
	origin := cube.Pos{0, 64, 0}

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		placed := assembler.Assemble("start", origin, 2, int64(i))
		assembler.BuildChunk(c, world.ChunkPos{0, 0}, placed, reg)
	}
}

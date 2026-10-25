package structure_test

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func TestTransformPos(t *testing.T) {
	size := cube.Pos{4, 5, 6}

	// 1. Rotation 0
	p0 := structure.TransformPos(cube.Pos{1, 2, 3}, size, structure.Rotation0, structure.MirrorNone)
	if p0 != (cube.Pos{1, 2, 3}) {
		t.Fatalf("expected (1, 2, 3), got %v", p0)
	}

	// 2. Rotation 90
	p90 := structure.TransformPos(cube.Pos{1, 2, 3}, size, structure.Rotation90, structure.MirrorNone)
	// (sz - 1 - z, y, x) = (6 - 1 - 3, 2, 1) = (2, 2, 1)
	if p90 != (cube.Pos{2, 2, 1}) {
		t.Fatalf("expected (2, 2, 1), got %v", p90)
	}

	// 3. Rotation 180
	p180 := structure.TransformPos(cube.Pos{1, 2, 3}, size, structure.Rotation180, structure.MirrorNone)
	// (sx - 1 - x, y, sz - 1 - z) = (4 - 1 - 1, 2, 6 - 1 - 3) = (2, 2, 2)
	if p180 != (cube.Pos{2, 2, 2}) {
		t.Fatalf("expected (2, 2, 2), got %v", p180)
	}

	// 4. Mirror X
	pmx := structure.TransformPos(cube.Pos{1, 2, 3}, size, structure.Rotation0, structure.MirrorX)
	// (sx - 1 - x, y, z) = (4 - 1 - 1, 2, 3) = (2, 2, 3)
	if pmx != (cube.Pos{2, 2, 3}) {
		t.Fatalf("expected (2, 2, 3), got %v", pmx)
	}
}

func TestTemplateBoundsAndNilSafety(t *testing.T) {
	// Negative dimensions
	negTmpl := structure.NewTemplate(cube.Pos{-1, 0, 5})
	if negTmpl.Dimensions() != (cube.Pos{0, 0, 0}) {
		t.Fatalf("expected 0 dimensions for negative size, got %v", negTmpl.Dimensions())
	}

	// Oversized dimensions (OOM defense)
	bombTmpl := structure.NewTemplate(cube.Pos{10000, 10000, 10000})
	if bombTmpl.Dimensions() != (cube.Pos{0, 0, 0}) {
		t.Fatalf("expected 0 dimensions for oversized bomb template, got %v", bombTmpl.Dimensions())
	}

	// Nil template operations
	var nilTmpl *structure.Template
	if nilTmpl.Dimensions() != (cube.Pos{0, 0, 0}) {
		t.Fatalf("expected 0 dimensions on nil template")
	}
	if b, ok := nilTmpl.Block(cube.Pos{0, 0, 0}); ok || b != nil {
		t.Fatalf("expected (nil, false) on nil template block lookup")
	}
	nilTmpl.SetBlock(cube.Pos{0, 0, 0}, block.Stone{})
	nilTmpl.Build(nil, cube.Pos{0, 0, 0}, structure.Rotation0, structure.MirrorNone)
	nilTmpl.BuildChunk(nil, world.ChunkPos{0, 0}, cube.Pos{0, 0, 0}, structure.Rotation0, structure.MirrorNone, nil)

	if _, err := structure.ToMCStructure(nil); err == nil {
		t.Fatalf("expected error serializing nil template to mcstructure")
	}
}

func TestMCStructureRoundTrip(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	stone := block.Stone{}
	dirt := block.Dirt{}
	grass := block.Grass{}

	tmpl := structure.NewTemplate(cube.Pos{3, 3, 3})
	tmpl.SetBlock(cube.Pos{0, 0, 0}, stone)
	tmpl.SetBlock(cube.Pos{1, 1, 1}, dirt)
	tmpl.SetBlock(cube.Pos{2, 2, 2}, grass)

	// Serialize to .mcstructure
	data, err := structure.ToMCStructure(tmpl)
	if err != nil {
		t.Fatalf("failed to serialize mcstructure: %v", err)
	}

	// Deserialize back
	parsed, err := structure.FromMCStructure(data, reg)
	if err != nil {
		t.Fatalf("failed to parse mcstructure: %v", err)
	}

	if parsed.Dimensions() != tmpl.Dimensions() {
		t.Fatalf("dimensions mismatch: got %v, expected %v", parsed.Dimensions(), tmpl.Dimensions())
	}

	b0, ok := parsed.Block(cube.Pos{0, 0, 0})
	if !ok || b0 != stone {
		t.Fatalf("expected stone at (0, 0, 0), got %v", b0)
	}

	b1, ok := parsed.Block(cube.Pos{1, 1, 1})
	if !ok || b1 != dirt {
		t.Fatalf("expected dirt at (1, 1, 1), got %v", b1)
	}

	b2, ok := parsed.Block(cube.Pos{2, 2, 2})
	if !ok || b2 != grass {
		t.Fatalf("expected grass at (2, 2, 2), got %v", b2)
	}
}

func TestJavaNBTParsing(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	// Construct synthetic Java NBT structure
	type palEntry struct {
		Name string `nbt:"Name"`
	}
	type blkEntry struct {
		Pos   []int32 `nbt:"pos"`
		State int32   `nbt:"state"`
	}
	type javaNBT struct {
		Size    []int32    `nbt:"size"`
		Palette []palEntry `nbt:"palette"`
		Blocks  []blkEntry `nbt:"blocks"`
	}

	raw := javaNBT{
		Size: []int32{2, 2, 2},
		Palette: []palEntry{
			{Name: "minecraft:stone"},
			{Name: "minecraft:dirt"},
		},
		Blocks: []blkEntry{
			{Pos: []int32{0, 0, 0}, State: 0},
			{Pos: []int32{1, 1, 1}, State: 1},
		},
	}

	buf := new(bytes.Buffer)
	gz := gzip.NewWriter(buf)
	if err := nbt.NewEncoderWithEncoding(gz, nbt.BigEndian).Encode(raw); err != nil {
		t.Fatalf("failed to encode synthetic java nbt: %v", err)
	}
	_ = gz.Close()

	tmpl, err := structure.FromJavaNBT(buf.Bytes(), reg)
	if err != nil {
		t.Fatalf("failed to parse java nbt: %v", err)
	}

	if tmpl.Dimensions() != (cube.Pos{2, 2, 2}) {
		t.Fatalf("unexpected dimensions: %v", tmpl.Dimensions())
	}

	b0, ok := tmpl.Block(cube.Pos{0, 0, 0})
	if !ok || b0 != (block.Stone{}) {
		t.Fatalf("expected stone at (0, 0, 0), got %v", b0)
	}

	b1, ok := tmpl.Block(cube.Pos{1, 1, 1})
	if !ok || b1 != (block.Dirt{}) {
		t.Fatalf("expected dirt at (1, 1, 1), got %v", b1)
	}
}

func TestBuildChunk(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	tmpl := structure.NewTemplate(cube.Pos{2, 2, 2})
	tmpl.SetBlock(cube.Pos{0, 0, 0}, block.Stone{})
	tmpl.SetBlock(cube.Pos{1, 1, 1}, block.Dirt{})

	c := chunk.New(reg, cube.Range{0, 255})
	tmpl.BuildChunk(c, world.ChunkPos{0, 0}, cube.Pos{4, 64, 4}, structure.Rotation0, structure.MirrorNone, reg)

	stoneRID := reg.BlockRuntimeID(block.Stone{})
	dirtRID := reg.BlockRuntimeID(block.Dirt{})

	if b := c.Block(4, 64, 4, 0); b != stoneRID {
		t.Fatalf("expected stone at (4, 64, 4), got %d", b)
	}
	if b := c.Block(5, 65, 5, 0); b != dirtRID {
		t.Fatalf("expected dirt at (5, 65, 5), got %d", b)
	}
}

func BenchmarkBuildChunk(b *testing.B) {
	world.DefaultBlockRegistry.Finalize()
	reg := world.DefaultBlockRegistry

	tmpl := structure.NewTemplate(cube.Pos{8, 8, 8})
	for x := range 8 {
		for y := range 8 {
			for z := range 8 {
				tmpl.SetBlock(cube.Pos{x, y, z}, block.Stone{})
			}
		}
	}

	c := chunk.New(reg, cube.Range{0, 255})
	origin := cube.Pos{2, 64, 2}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		tmpl.BuildChunk(c, world.ChunkPos{0, 0}, origin, structure.Rotation0, structure.MirrorNone, reg)
	}
}

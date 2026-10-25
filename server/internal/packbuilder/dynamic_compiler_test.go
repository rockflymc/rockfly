package packbuilder

import (
	"image"
	"image/color"
	"testing"
)

func TestDynamicCompiler_InMemoryCompile(t *testing.T) {
	compiler := NewDynamicCompiler()

	compiler.AddText("textures/test.txt", "hello bedrock")
	compiler.AddLang("tile.rockfly:ruby_block.name", "Ruby Block")

	pack, hash, err := compiler.Compile("Rockfly Dynamic Pack", "Auto-generated test pack")
	if err != nil {
		t.Fatalf("failed to compile pack: %v", err)
	}

	if pack == nil {
		t.Fatalf("expected non-nil pack")
	}

	if pack.Name() != "Rockfly Dynamic Pack" {
		t.Fatalf("expected pack name 'Rockfly Dynamic Pack', got %s", pack.Name())
	}

	if hash == [32]byte{} {
		t.Fatalf("expected non-zero SHA-256 hash")
	}

	// Verify ReadFile works on compiled pack
	lang, err := pack.ReadFile("texts/en_US.lang")
	if err != nil || len(lang) == 0 {
		t.Fatalf("failed to read texts/en_US.lang from pack: %v", err)
	}
}

func TestDynamicCompiler_CustomBlockRegistration(t *testing.T) {
	compiler := NewDynamicCompiler()

	// 16x16 solid red texture
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	blk := CustomBlockData{
		Name: "Ruby Ore",
		Textures: map[string]image.Image{
			"ruby_ore": img,
		},
		Geometry: []byte(`{"format_version":"1.12.0","minecraft:geometry":[{"description":{"identifier":"geometry.ruby_ore"}}]}`),
	}

	if err := compiler.AddBlockData("rockfly:ruby_ore", blk); err != nil {
		t.Fatalf("failed to add custom block: %v", err)
	}

	pack, _, err := compiler.Compile("Custom Content Pack", "Testing custom blocks")
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	// Verify files via pack.ReadFile
	if _, err := pack.ReadFile("manifest.json"); err != nil {
		t.Fatalf("missing manifest.json: %v", err)
	}
	if _, err := pack.ReadFile("textures/terrain_texture.json"); err != nil {
		t.Fatalf("missing textures/terrain_texture.json: %v", err)
	}
	if _, err := pack.ReadFile("textures/blocks/ruby_ore.png"); err != nil {
		t.Fatalf("missing textures/blocks/ruby_ore.png: %v", err)
	}
	if _, err := pack.ReadFile("texts/en_US.lang"); err != nil {
		t.Fatalf("missing texts/en_US.lang: %v", err)
	}
	if _, err := pack.ReadFile("models/blocks/ruby_ore.geo.json"); err != nil {
		t.Fatalf("missing models/blocks/ruby_ore.geo.json: %v", err)
	}
}

func BenchmarkDynamicCompiler_Compile(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c := NewDynamicCompiler()
		_ = c.AddBlockData("rockfly:test_block", CustomBlockData{
			Name:     "Test Block",
			Textures: map[string]image.Image{"test": img},
		})
		_, _, _ = c.Compile("Benchmark Pack", "Description")
	}
}

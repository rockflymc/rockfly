package structure

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type javaPaletteEntry struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties,omitempty"`
}

type javaBlockPlacement struct {
	Pos   []int32        `nbt:"pos"`
	State int32          `nbt:"state"`
	NBT   map[string]any `nbt:"nbt,omitempty"`
}

type javaStructureData struct {
	DataVersion int32                `nbt:"DataVersion,omitempty"`
	Size        []int32              `nbt:"size"`
	Palette     []javaPaletteEntry   `nbt:"palette"`
	Blocks      []javaBlockPlacement `nbt:"blocks"`
	Entities    []any                `nbt:"entities,omitempty"`
}

// FromJavaNBT parses a raw or gzipped Java Edition structure .nbt buffer into a Template.
func FromJavaNBT(data []byte, reg world.BlockRegistry) (*Template, error) {
	// Decompress gzip if present (0x1f, 0x8b)
	reader := bytes.NewReader(data)
	var r io.Reader = reader
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gz.Close()
		r = gz
	}

	var jData javaStructureData
	if err := nbt.NewDecoderWithEncoding(r, nbt.BigEndian).Decode(&jData); err != nil {
		return nil, fmt.Errorf("failed to decode java nbt structure: %w", err)
	}

	if len(jData.Size) != 3 {
		return nil, fmt.Errorf("invalid size field in java structure: %v", jData.Size)
	}

	size := cube.Pos{int(jData.Size[0]), int(jData.Size[1]), int(jData.Size[2])}
	t := NewTemplate(size)

	// Resolve palette
	palette := make([]world.Block, len(jData.Palette))
	for i, entry := range jData.Palette {
		props := make(map[string]any, len(entry.Properties))
		for k, v := range entry.Properties {
			props[k] = v
		}
		b, ok := reg.BlockByName(entry.Name, props)
		if !ok {
			b, ok = reg.BlockByName(entry.Name, nil)
		}
		if ok {
			palette[i] = b
		}
	}

	// Populate blocks
	for _, blk := range jData.Blocks {
		if len(blk.Pos) != 3 {
			continue
		}
		pos := cube.Pos{int(blk.Pos[0]), int(blk.Pos[1]), int(blk.Pos[2])}
		if blk.State >= 0 && int(blk.State) < len(palette) && palette[blk.State] != nil {
			t.SetBlock(pos, palette[blk.State])
		}
	}

	return t, nil
}

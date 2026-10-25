package structure

import (
	"bytes"
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type mcBlockEntry struct {
	Name    string         `nbt:"name"`
	States  map[string]any `nbt:"states"`
	Version int32          `nbt:"version"`
}

type mcDefaultPalette struct {
	BlockPalette      []mcBlockEntry `nbt:"block_palette"`
	BlockPositionData map[string]any `nbt:"block_position_data,omitempty"`
}

type mcStructurePalette struct {
	Default mcDefaultPalette `nbt:"default"`
}

type mcStructureField struct {
	BlockIndices [][]int32          `nbt:"block_indices"`
	Entities     []map[string]any   `nbt:"entities"`
	Palette      mcStructurePalette `nbt:"palette"`
}

type mcStructureData struct {
	FormatVersion        int32            `nbt:"format_version"`
	Size                 []int32          `nbt:"size"`
	Structure            mcStructureField `nbt:"structure"`
	StructureWorldOrigin []int32          `nbt:"structure_world_origin,omitempty"`
}

// FromMCStructure parses a raw Bedrock Edition .mcstructure binary buffer into a Template.
func FromMCStructure(data []byte, reg world.BlockRegistry) (*Template, error) {
	var sData mcStructureData
	if err := nbt.NewDecoderWithEncoding(bytes.NewBuffer(data), nbt.LittleEndian).Decode(&sData); err != nil {
		return nil, fmt.Errorf("failed to decode mcstructure nbt: %w", err)
	}

	if len(sData.Size) != 3 {
		return nil, fmt.Errorf("invalid size field in mcstructure: %v", sData.Size)
	}

	size := cube.Pos{int(sData.Size[0]), int(sData.Size[1]), int(sData.Size[2])}
	t := NewTemplate(size)

	// Resolve block palette
	palette := make([]world.Block, len(sData.Structure.Palette.Default.BlockPalette))
	for i, entry := range sData.Structure.Palette.Default.BlockPalette {
		b, ok := reg.BlockByName(entry.Name, entry.States)
		if !ok {
			// Fallback: lookup by name without states
			b, ok = reg.BlockByName(entry.Name, nil)
		}
		if ok {
			palette[i] = b
		}
	}

	if len(sData.Structure.BlockIndices) == 0 {
		return t, nil
	}

	// Layer 0 is primary block layer
	primary := sData.Structure.BlockIndices[0]
	expectedVolume := size.X() * size.Y() * size.Z()

	for idx := 0; idx < len(primary) && idx < expectedVolume; idx++ {
		pIdx := primary[idx]
		if pIdx >= 0 && int(pIdx) < len(palette) && palette[pIdx] != nil {
			// MCStructure flat indexing: (x * size.Y + y) * size.Z + z
			x := idx / (size.Y() * size.Z())
			rem := idx % (size.Y() * size.Z())
			y := rem / size.Z()
			z := rem % size.Z()

			t.SetBlock(cube.Pos{x, y, z}, palette[pIdx])
		}
	}

	return t, nil
}

// ToMCStructure serializes a Template into canonical Bedrock Edition .mcstructure format.
func ToMCStructure(t *Template) ([]byte, error) {
	size := t.Dimensions()
	volume := size.X() * size.Y() * size.Z()

	blockPalette := make([]mcBlockEntry, len(t.palette))
	for i, b := range t.palette {
		name, states := b.EncodeBlock()
		blockPalette[i] = mcBlockEntry{
			Name:    name,
			States:  states,
			Version: 18090528, // Bedrock 1.20+ data version
		}
	}

	primaryIndices := make([]int32, volume)
	secondaryIndices := make([]int32, volume)

	for x := range size.X() {
		for y := range size.Y() {
			for z := range size.Z() {
				idx := (x*size.Y()+y)*size.Z() + z
				pIdx := t.indices[idx]
				primaryIndices[idx] = pIdx
				secondaryIndices[idx] = -1
			}
		}
	}

	sData := mcStructureData{
		FormatVersion: 1,
		Size:          []int32{int32(size.X()), int32(size.Y()), int32(size.Z())},
		Structure: mcStructureField{
			BlockIndices: [][]int32{primaryIndices, secondaryIndices},
			Palette: mcStructurePalette{
				Default: mcDefaultPalette{
					BlockPalette: blockPalette,
				},
			},
		},
	}

	buf := new(bytes.Buffer)
	if err := nbt.NewEncoderWithEncoding(buf, nbt.LittleEndian).Encode(sData); err != nil {
		return nil, fmt.Errorf("failed to encode mcstructure: %w", err)
	}

	return buf.Bytes(), nil
}

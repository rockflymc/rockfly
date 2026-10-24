package surface

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// BadlandsBands manages the canonical 1:1 terracotta strata layers found in Badlands/Mesa biomes.
type BadlandsBands struct {
	layers [64]uint32
}

// NewBadlandsBands creates a BadlandsBands palette using the specified BlockRegistry.
func NewBadlandsBands(reg world.BlockRegistry) *BadlandsBands {
	b := &BadlandsBands{}

	undyed := reg.BlockRuntimeID(block.Terracotta{})
	white := reg.BlockRuntimeID(block.StainedTerracotta{Colour: item.ColourWhite()})
	orange := reg.BlockRuntimeID(block.StainedTerracotta{Colour: item.ColourOrange()})
	yellow := reg.BlockRuntimeID(block.StainedTerracotta{Colour: item.ColourYellow()})
	brown := reg.BlockRuntimeID(block.StainedTerracotta{Colour: item.ColourBrown()})
	red := reg.BlockRuntimeID(block.StainedTerracotta{Colour: item.ColourRed()})
	lightGrey := reg.BlockRuntimeID(block.StainedTerracotta{Colour: item.ColourLightGrey()})

	// Canonical 64-height sequence repeating across the vertical Mesa landscape
	pattern := []uint32{
		orange, orange, undyed, undyed, undyed, brown, brown, undyed,
		white, white, lightGrey, orange, orange, undyed, undyed, undyed,
		yellow, yellow, undyed, brown, brown, red, red, undyed,
		white, orange, orange, undyed, undyed, undyed, undyed, brown,
		undyed, undyed, lightGrey, orange, orange, undyed, undyed, red,
		orange, white, white, undyed, yellow, yellow, undyed, undyed,
		brown, brown, undyed, undyed, undyed, red, red, undyed,
		orange, orange, white, undyed, undyed, brown, undyed, lightGrey,
	}

	copy(b.layers[:], pattern)
	return b
}

// BandAt returns the terracotta block runtime ID at absolute height y.
func (b *BadlandsBands) BandAt(y int) uint32 {
	idx := (y%64 + 64) % 64
	return b.layers[idx]
}

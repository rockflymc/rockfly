package starlight

import (
	"github.com/df-mc/dragonfly/server/block/cube"
)

// YOffset is the vertical offset added to Y coordinates so that negative coordinates
// down to -512 can be encoded in positive unsigned bits.
const YOffset = 512

// PackedNode represents a light propagation node packed into a single 64-bit integer.
//
// Bit layout:
// - Bits  0..5  (6 bits):  X coordinate [0..63]
// - Bits  6..16 (11 bits): Y coordinate with YOffset [0..2047] (supports Y from -512 to 1535)
// - Bits 17..22 (6 bits):  Z coordinate [0..63]
// - Bits 23..26 (4 bits):  Light Level [0..15]
// - Bit  27     (1 bit):   Light Type (0 = BlockLight, 1 = SkyLight)
// - Bits 28..31 (4 bits):  Flags (e.g. removal/spread mode)
type PackedNode uint64

const (
	maskX     = 0x3f  // 6 bits (0..63)
	maskY     = 0x7ff // 11 bits (0..2047)
	maskZ     = 0x3f  // 6 bits (0..63)
	maskLevel = 0x0f  // 4 bits (0..15)
	flagSky   = PackedNode(1) << 27
)

// PackNode packs coordinates, light level, and light type into a PackedNode.
func PackNode(x, y, z int, level uint8, isSky bool) PackedNode {
	v := PackedNode(x&maskX) |
		(PackedNode((y+YOffset)&maskY) << 6) |
		(PackedNode(z&maskZ) << 17) |
		(PackedNode(level&maskLevel) << 23)

	if isSky {
		v |= flagSky
	}
	return v
}

// X returns the X coordinate of the node.
func (n PackedNode) X() int {
	return int(n & maskX)
}

// Y returns the Y coordinate of the node, subtracting YOffset.
func (n PackedNode) Y() int {
	return int((n>>6)&maskY) - YOffset
}

// Z returns the Z coordinate of the node.
func (n PackedNode) Z() int {
	return int((n >> 17) & maskZ)
}

// Level returns the light level [0..15] of the node.
func (n PackedNode) Level() uint8 {
	return uint8((n >> 23) & maskLevel)
}

// IsSky returns true if this is a SkyLight node, false if BlockLight.
func (n PackedNode) IsSky() bool {
	return (n & flagSky) != 0
}

// Pos returns the absolute world position of the node given the base coordinate of the area.
func (n PackedNode) Pos(baseX, baseZ int) cube.Pos {
	return cube.Pos{baseX + n.X(), n.Y(), baseZ + n.Z()}
}

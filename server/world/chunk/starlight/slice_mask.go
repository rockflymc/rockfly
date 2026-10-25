package starlight

import (
	"math/bits"
)

// SliceMask represents a 256-bit bitmask for a 16x16 horizontal layer of blocks in a subchunk.
// Each bit corresponds to block (x, z) with index (z << 4) | x.
// 4 x 64-bit integers represent the entire 256-block plane with zero heap allocations.
type SliceMask [4]uint64

// Set marks block at (x, z) in the slice.
func (m *SliceMask) Set(x, z int) {
	idx := (z << 4) | (x & 0x0f)
	m[idx>>6] |= uint64(1) << (idx & 0x3f)
}

// Clear unmarks block at (x, z) in the slice.
func (m *SliceMask) Clear(x, z int) {
	idx := (z << 4) | (x & 0x0f)
	m[idx>>6] &^= uint64(1) << (idx & 0x3f)
}

// Get checks whether block at (x, z) is marked in the slice.
func (m SliceMask) Get(x, z int) bool {
	idx := (z << 4) | (x & 0x0f)
	return (m[idx>>6] & (uint64(1) << (idx & 0x3f))) != 0
}

// Empty returns true if no blocks are marked in this slice (e.g. completely transparent air).
func (m SliceMask) Empty() bool {
	return m[0] == 0 && m[1] == 0 && m[2] == 0 && m[3] == 0
}

// Full returns true if all 256 blocks are marked (e.g. completely solid stone).
func (m SliceMask) Full() bool {
	return m[0] == ^uint64(0) && m[1] == ^uint64(0) && m[2] == ^uint64(0) && m[3] == ^uint64(0)
}

// Count returns the number of set bits in this slice (0..256).
func (m SliceMask) Count() int {
	return bits.OnesCount64(m[0]) + bits.OnesCount64(m[1]) + bits.OnesCount64(m[2]) + bits.OnesCount64(m[3])
}

// And computes the bitwise AND of two masks.
func (m SliceMask) And(other SliceMask) SliceMask {
	return SliceMask{
		m[0] & other[0],
		m[1] & other[1],
		m[2] & other[2],
		m[3] & other[3],
	}
}

// Or computes the bitwise OR of two masks.
func (m SliceMask) Or(other SliceMask) SliceMask {
	return SliceMask{
		m[0] | other[0],
		m[1] | other[1],
		m[2] | other[2],
		m[3] | other[3],
	}
}

// Not computes the bitwise inversion of this mask.
func (m SliceMask) Not() SliceMask {
	return SliceMask{
		^m[0],
		^m[1],
		^m[2],
		^m[3],
	}
}

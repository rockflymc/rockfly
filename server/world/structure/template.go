package structure

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// Template represents an in-memory 3D voxel structure template with an indexed block palette.
type Template struct {
	size    cube.Pos
	palette []world.Block
	indices []int32
}

// NewTemplate constructs an empty Template with the specified bounding box dimensions.
func NewTemplate(size cube.Pos) *Template {
	volume := size.X() * size.Y() * size.Z()
	indices := make([]int32, volume)
	for i := range indices {
		indices[i] = -1 // -1 signifies unpopulated / transparent void
	}
	return &Template{
		size:    size,
		palette: nil,
		indices: indices,
	}
}

// Dimensions returns the (width, height, depth) dimensions of the template.
func (t *Template) Dimensions() cube.Pos {
	return t.size
}

// index returns the 1D flat index for coordinate (x, y, z).
func (t *Template) index(x, y, z int) int {
	return (x*t.size.Y()+y)*t.size.Z() + z
}

// SetBlock sets the block at local coordinates (x, y, z).
func (t *Template) SetBlock(pos cube.Pos, b world.Block) {
	if pos.X() < 0 || pos.X() >= t.size.X() ||
		pos.Y() < 0 || pos.Y() >= t.size.Y() ||
		pos.Z() < 0 || pos.Z() >= t.size.Z() {
		return
	}

	if b == nil || (b == block.Air{}) {
		t.indices[t.index(pos.X(), pos.Y(), pos.Z())] = -1
		return
	}

	// Find or register in palette using block hash
	bH1, bH2 := b.Hash()
	var pIdx int32 = -1
	for i, palBlock := range t.palette {
		pH1, pH2 := palBlock.Hash()
		if pH1 == bH1 && pH2 == bH2 {
			pIdx = int32(i)
			break
		}
	}
	if pIdx == -1 {
		pIdx = int32(len(t.palette))
		t.palette = append(t.palette, b)
	}

	t.indices[t.index(pos.X(), pos.Y(), pos.Z())] = pIdx
}

// Block returns the block at local coordinates pos.
func (t *Template) Block(pos cube.Pos) (world.Block, bool) {
	if pos.X() < 0 || pos.X() >= t.size.X() ||
		pos.Y() < 0 || pos.Y() >= t.size.Y() ||
		pos.Z() < 0 || pos.Z() >= t.size.Z() {
		return nil, false
	}
	pIdx := t.indices[t.index(pos.X(), pos.Y(), pos.Z())]
	if pIdx < 0 || int(pIdx) >= len(t.palette) {
		return nil, false
	}
	return t.palette[pIdx], true
}

// Build places the template blocks into the world transaction tx at world origin after applying rotation and mirroring.
func (t *Template) Build(tx *world.Tx, origin cube.Pos, rot Rotation, mirror Mirror) {
	for x := range t.size.X() {
		for y := range t.size.Y() {
			for z := range t.size.Z() {
				pIdx := t.indices[t.index(x, y, z)]
				if pIdx < 0 || int(pIdx) >= len(t.palette) {
					continue
				}

				b := t.palette[pIdx]
				dstPos := TransformPos(cube.Pos{x, y, z}, t.size, rot, mirror).Add(origin)
				tx.SetBlock(dstPos, b, nil)
			}
		}
	}
}

// BuildChunk places template blocks directly into a generating chunk column for zero-allocation performance.
func (t *Template) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, origin cube.Pos, rot Rotation, mirror Mirror, reg world.BlockRegistry) {
	minChunkX := int(chunkPos.X()) << 4
	maxChunkX := minChunkX + 15
	minChunkZ := int(chunkPos.Z()) << 4
	maxChunkZ := minChunkZ + 15

	// Pre-resolve palette runtime IDs
	runtimeIDs := make([]uint32, len(t.palette))
	for i, b := range t.palette {
		runtimeIDs[i] = reg.BlockRuntimeID(b)
	}

	for x := range t.size.X() {
		for y := range t.size.Y() {
			for z := range t.size.Z() {
				pIdx := t.indices[t.index(x, y, z)]
				if pIdx < 0 || int(pIdx) >= len(t.palette) {
					continue
				}

				dstPos := TransformPos(cube.Pos{x, y, z}, t.size, rot, mirror).Add(origin)
				wx, wy, wz := dstPos.X(), dstPos.Y(), dstPos.Z()

				// Check if within this chunk's horizontal bounds
				if wx >= minChunkX && wx <= maxChunkX && wz >= minChunkZ && wz <= maxChunkZ {
					lx := uint8(wx - minChunkX)
					lz := uint8(wz - minChunkZ)
					c.SetBlock(lx, int16(wy), lz, 0, runtimeIDs[pIdx])
				}
			}
		}
	}
}

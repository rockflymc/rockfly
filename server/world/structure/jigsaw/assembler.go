package jigsaw

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// Assembler manages structure pools and executes recursive jigsaw connection graph assembly.
type Assembler struct {
	pools map[string]*Pool
}

// NewAssembler creates an empty Jigsaw Assembler.
func NewAssembler() *Assembler {
	return &Assembler{
		pools: make(map[string]*Pool),
	}
}

// RegisterPool registers a structure pool by name.
func (a *Assembler) RegisterPool(p *Pool) {
	a.pools[p.Name] = p
}

type openConnection struct {
	parentPiece *PlacedPiece
	jigsaw      JigsawBlock
	depth       int
}

// RotateFace rotates a cube.Face clockwise around the Y axis by the given rotation.
func RotateFace(f cube.Face, rot structure.Rotation) cube.Face {
	if f == cube.FaceUp || f == cube.FaceDown {
		return f
	}
	steps := int(rot)
	for range steps {
		switch f {
		case cube.FaceNorth:
			f = cube.FaceEast
		case cube.FaceEast:
			f = cube.FaceSouth
		case cube.FaceSouth:
			f = cube.FaceWest
		case cube.FaceWest:
			f = cube.FaceNorth
		}
	}
	return f
}

// Assemble builds a multi-piece structure graph starting from startPool at origin.
func (a *Assembler) Assemble(startPool string, origin cube.Pos, maxDepth int, seed int64) []*PlacedPiece {
	pool, ok := a.pools[startPool]
	if !ok || len(pool.Pieces) == 0 {
		return nil
	}

	startPiece := pool.Select(int(seed))
	if startPiece == nil {
		return nil
	}

	startBox := NewBoundingBox(origin, startPiece.Template.Dimensions())
	placed := []*PlacedPiece{
		{
			Piece:    startPiece,
			Origin:   origin,
			Rotation: structure.Rotation0,
			Box:      startBox,
		},
	}

	var queue []openConnection
	for _, j := range startPiece.Jigsaws {
		queue = append(queue, openConnection{
			parentPiece: placed[0],
			jigsaw:      j,
			depth:       1,
		})
	}

	allRotations := []structure.Rotation{
		structure.Rotation0,
		structure.Rotation90,
		structure.Rotation180,
		structure.Rotation270,
	}

	counter := 0
	for len(queue) > 0 {
		conn := queue[0]
		queue = queue[1:]

		if conn.depth > maxDepth {
			continue
		}

		targetPoolName := conn.jigsaw.Target
		targetPool, poolFound := a.pools[targetPoolName]
		if !poolFound {
			targetPool, poolFound = a.pools[conn.jigsaw.Pool]
		}
		if !poolFound || len(targetPool.Pieces) == 0 {
			continue
		}

		counter++
		candPiece := targetPool.Select(int(seed) + counter*31 + conn.depth*17)
		if candPiece == nil {
			continue
		}

		parentFace := RotateFace(conn.jigsaw.Facing, conn.parentPiece.Rotation)
		requiredChildFace := parentFace.Opposite()

		parentWorldJigsaw := conn.parentPiece.Origin.Add(
			structure.TransformPos(conn.jigsaw.Pos, conn.parentPiece.Piece.Template.Dimensions(), conn.parentPiece.Rotation, structure.MirrorNone),
		)
		childTargetWorldPos := parentWorldJigsaw.Side(parentFace)

		var success bool
		for childJIdx, childJ := range candPiece.Jigsaws {
			if success {
				break
			}
			// Match connection target
			if childJ.Target != conn.jigsaw.Name && childJ.Name != conn.jigsaw.Target && conn.jigsaw.Target != targetPoolName {
				continue
			}

			for _, rot := range allRotations {
				if RotateFace(childJ.Facing, rot) != requiredChildFace {
					continue
				}

				childTransformedLocal := structure.TransformPos(childJ.Pos, candPiece.Template.Dimensions(), rot, structure.MirrorNone)
				childOrigin := childTargetWorldPos.Sub(childTransformedLocal)
				childDim := structure.TransformDimensions(candPiece.Template.Dimensions(), rot)
				childBox := NewBoundingBox(childOrigin, childDim)

				// Check collision against all placed pieces
				collides := false
				for _, p := range placed {
					if childBox.Intersects(p.Box) {
						collides = true
						break
					}
				}

				if !collides {
					newPlaced := &PlacedPiece{
						Piece:    candPiece,
						Origin:   childOrigin,
						Rotation: rot,
						Box:      childBox,
					}
					placed = append(placed, newPlaced)
					success = true

					// Enqueue open jigsaws on the new piece (excluding the one just connected)
					for otherIdx, otherJ := range candPiece.Jigsaws {
						if otherIdx == childJIdx {
							continue
						}
						queue = append(queue, openConnection{
							parentPiece: newPlaced,
							jigsaw:      otherJ,
							depth:       conn.depth + 1,
						})
					}
					break
				}
			}
		}
	}

	return placed
}

// Build places all assembled pieces into a world transaction and replaces jigsaws with FinalBlock.
func (a *Assembler) Build(tx *world.Tx, pieces []*PlacedPiece) {
	for _, p := range pieces {
		p.Piece.Template.Build(tx, p.Origin, p.Rotation, structure.MirrorNone)
		for _, j := range p.Piece.Jigsaws {
			if j.FinalBlock != nil {
				pos := p.Origin.Add(structure.TransformPos(j.Pos, p.Piece.Template.Dimensions(), p.Rotation, structure.MirrorNone))
				tx.SetBlock(pos, j.FinalBlock, nil)
			}
		}
	}
}

// BuildChunk populates pieces overlapping the specified chunk column for zero-allocation generation.
func (a *Assembler) BuildChunk(c *chunk.Chunk, chunkPos world.ChunkPos, pieces []*PlacedPiece, reg world.BlockRegistry) {
	minChunkX := int(chunkPos.X()) << 4
	maxChunkX := minChunkX + 15
	minChunkZ := int(chunkPos.Z()) << 4
	maxChunkZ := minChunkZ + 15

	chunkBox := BoundingBox{
		Min: cube.Pos{minChunkX, -64, minChunkZ},
		Max: cube.Pos{maxChunkX, 319, maxChunkZ},
	}

	for _, p := range pieces {
		if !chunkBox.Intersects(p.Box) {
			continue
		}
		p.Piece.Template.BuildChunk(c, chunkPos, p.Origin, p.Rotation, structure.MirrorNone, reg)
		for _, j := range p.Piece.Jigsaws {
			if j.FinalBlock != nil {
				pos := p.Origin.Add(structure.TransformPos(j.Pos, p.Piece.Template.Dimensions(), p.Rotation, structure.MirrorNone))
				if pos.X() >= minChunkX && pos.X() <= maxChunkX && pos.Z() >= minChunkZ && pos.Z() <= maxChunkZ {
					c.SetBlock(uint8(pos.X()-minChunkX), int16(pos.Y()), uint8(pos.Z()-minChunkZ), 0, reg.BlockRuntimeID(j.FinalBlock))
				}
			}
		}
	}
}

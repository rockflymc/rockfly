package jigsaw

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/structure"
)

// BoundingBox represents an integer axis-aligned bounding box (AABB) in world coordinate space.
type BoundingBox struct {
	Min cube.Pos
	Max cube.Pos
}

// NewBoundingBox creates an inclusive AABB from origin and dimensions.
func NewBoundingBox(origin, size cube.Pos) BoundingBox {
	return BoundingBox{
		Min: origin,
		Max: cube.Pos{
			origin.X() + size.X() - 1,
			origin.Y() + size.Y() - 1,
			origin.Z() + size.Z() - 1,
		},
	}
}

// Intersects checks if this bounding box overlaps with another bounding box.
func (b BoundingBox) Intersects(other BoundingBox) bool {
	return b.Min.X() <= other.Max.X() && b.Max.X() >= other.Min.X() &&
		b.Min.Y() <= other.Max.Y() && b.Max.Y() >= other.Min.Y() &&
		b.Min.Z() <= other.Max.Z() && b.Max.Z() >= other.Min.Z()
}

// Piece represents an unplaced structure element with its template and connection anchors.
type Piece struct {
	Template *structure.Template
	Jigsaws  []JigsawBlock
}

// PlacedPiece represents an instantiated, positioned, and oriented piece within an assembled structure.
type PlacedPiece struct {
	Piece    *Piece
	Origin   cube.Pos
	Rotation structure.Rotation
	Box      BoundingBox
}

package structure

import (
	"github.com/df-mc/dragonfly/server/block/cube"
)

// Rotation represents a 90-degree step clockwise rotation around the vertical Y axis.
type Rotation int

const (
	Rotation0 Rotation = iota
	Rotation90
	Rotation180
	Rotation270
)

// Mirror represents horizontal axis reflection.
type Mirror int

const (
	MirrorNone Mirror = iota
	MirrorX
	MirrorZ
)

// TransformDimensions computes the transformed bounding box size after applying rotation and mirroring.
func TransformDimensions(size cube.Pos, rot Rotation) cube.Pos {
	switch rot {
	case Rotation90, Rotation270:
		return cube.Pos{size.Z(), size.Y(), size.X()}
	default:
		return size
	}
}

// TransformPos calculates the destination coordinate of (x, y, z) inside a bounding box of size (sx, sy, sz)
// after applying horizontal mirroring and clockwise rotation around the Y axis.
func TransformPos(pos, size cube.Pos, rot Rotation, mirror Mirror) cube.Pos {
	x, y, z := pos.X(), pos.Y(), pos.Z()
	sx, sz := size.X(), size.Z()

	// 1. Mirroring
	switch mirror {
	case MirrorX:
		x = sx - 1 - x
	case MirrorZ:
		z = sz - 1 - z
	}

	// 2. Rotation around Y
	switch rot {
	case Rotation90:
		return cube.Pos{sz - 1 - z, y, x}
	case Rotation180:
		return cube.Pos{sx - 1 - x, y, sz - 1 - z}
	case Rotation270:
		return cube.Pos{z, y, sx - 1 - x}
	default: // Rotation0
		return cube.Pos{x, y, z}
	}
}

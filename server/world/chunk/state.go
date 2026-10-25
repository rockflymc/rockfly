package chunk

import (
	"fmt"
)

// Pos represents the coordinate of a Chunk in the world.
type Pos [2]int32

// X returns the X coordinate of the chunk.
func (p Pos) X() int32 { return p[0] }

// Z returns the Z coordinate of the chunk.
func (p Pos) Z() int32 { return p[1] }

// String returns a human-readable representation of Pos.
func (p Pos) String() string {
	return fmt.Sprintf("(%d, %d)", p[0], p[1])
}

// Distance returns the Chebyshev distance (maximum coordinate difference) between two chunk positions.
func (p Pos) Distance(other Pos) int32 {
	dx := p[0] - other[0]
	if dx < 0 {
		dx = -dx
	}
	dz := p[1] - other[1]
	if dz < 0 {
		dz = -dz
	}
	if dx > dz {
		return dx
	}
	return dz
}

// Chunk level constants following PaperMC/Tuinity 4-tier chunk ticking hierarchy.
// Lower numeric values represent higher priority/activity.
const (
	// LevelEntityTicking indicates that entities, block entities, redstone, liquids, and random ticks are active.
	LevelEntityTicking = 31
	// LevelTicking indicates that blocks, redstone, liquids, and scheduled ticks are active, but entities do not tick.
	LevelTicking = 32
	// LevelBorder indicates that the chunk is loaded, lighted, and visible to players, but neither blocks nor entities tick.
	LevelBorder = 33
	// LevelInaccessible indicates that the chunk is not ticking and is eligible for unloading once its linger timer expires.
	LevelInaccessible = 34
	// LevelMax is the absolute maximum level representing an unloaded/out-of-range chunk.
	LevelMax = 45
)

// ChunkState represents the 4-tier lifecycle state of a chunk.
type ChunkState uint8

const (
	// StateInaccessible means the chunk is inaccessible and awaiting unload.
	StateInaccessible ChunkState = iota
	// StateBorder means the chunk is loaded in memory and visible, but inactive.
	StateBorder
	// StateTicking means blocks, redstone, and scheduled ticks update, but not entities.
	StateTicking
	// StateEntityTicking means all gameplay systems including entities tick fully.
	StateEntityTicking
)

// StateFromLevel maps a numeric chunk ticket level to its corresponding ChunkState.
func StateFromLevel(level int) ChunkState {
	switch {
	case level <= LevelEntityTicking:
		return StateEntityTicking
	case level == LevelTicking:
		return StateTicking
	case level == LevelBorder:
		return StateBorder
	default:
		return StateInaccessible
	}
}

// String returns the name of the ChunkState.
func (s ChunkState) String() string {
	switch s {
	case StateEntityTicking:
		return "EntityTicking"
	case StateTicking:
		return "Ticking"
	case StateBorder:
		return "Border"
	case StateInaccessible:
		return "Inaccessible"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// CanTickBlocks returns whether blocks, liquids, and redstone in this chunk can tick.
func (s ChunkState) CanTickBlocks() bool {
	return s >= StateTicking
}

// CanTickEntities returns whether entities in this chunk can tick.
func (s ChunkState) CanTickEntities() bool {
	return s == StateEntityTicking
}

// IsLoaded returns whether the chunk is loaded into memory (Border or above).
func (s ChunkState) IsLoaded() bool {
	return s >= StateBorder
}

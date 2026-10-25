package generator

import (
	"errors"
	"fmt"
	"sync"

	"github.com/df-mc/dragonfly/server/world/chunk"
)

var (
	// ErrOutOfRegion is returned when attempting to access a coordinate outside the WorldGenRegion bounds.
	ErrOutOfRegion = errors.New("coordinate is outside the WorldGenRegion bounds")
	// ErrMissingChunk is returned when a chunk within the region radius has not been staged.
	ErrMissingChunk = errors.New("required chunk is missing from WorldGenRegion")
)

// WorldGenRegion represents a bounded window of ProtoChunks around a center chunk.
// It allows feature and structure generators to safely read and write across chunk boundaries
// (e.g., placing tree leaves or ore veins that cross borders) without triggering recursive
// chunk loading deadlocks.
type WorldGenRegion struct {
	mu          sync.RWMutex
	center      chunk.Pos
	radius      int
	chunks      map[chunk.Pos]*ProtoChunk
	targetStage GenerationStage
}

// NewWorldGenRegion constructs a new WorldGenRegion centered at center with the specified chunk radius.
// If radius is 1, the region encompasses a 3x3 grid of chunks.
func NewWorldGenRegion(center chunk.Pos, radius int, chunks map[chunk.Pos]*ProtoChunk, targetStage GenerationStage) (*WorldGenRegion, error) {
	if radius < 0 {
		return nil, fmt.Errorf("invalid region radius: %d", radius)
	}
	if chunks == nil {
		chunks = make(map[chunk.Pos]*ProtoChunk)
	}

	// Verify all chunks in the (2R+1)x(2R+1) square are present.
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			p := chunk.Pos{center[0] + int32(dx), center[1] + int32(dz)}
			if _, ok := chunks[p]; !ok {
				return nil, fmt.Errorf("%w: chunk at %v", ErrMissingChunk, p)
			}
		}
	}

	return &WorldGenRegion{
		center:      center,
		radius:      radius,
		chunks:      chunks,
		targetStage: targetStage,
	}, nil
}

// Center returns the center chunk position.
func (r *WorldGenRegion) Center() chunk.Pos {
	return r.center
}

// Radius returns the chunk radius.
func (r *WorldGenRegion) Radius() int {
	return r.radius
}

// TargetStage returns the stage this region is preparing chunks for.
func (r *WorldGenRegion) TargetStage() GenerationStage {
	return r.targetStage
}

// InRange returns true if the world coordinates (x, z) fall within this region's chunk window.
func (r *WorldGenRegion) InRange(worldX, worldZ int) bool {
	chunkX := int32(worldX >> 4)
	chunkZ := int32(worldZ >> 4)
	dx := chunkX - r.center[0]
	dz := chunkZ - r.center[1]
	if dx < 0 {
		dx = -dx
	}
	if dz < 0 {
		dz = -dz
	}
	return dx <= int32(r.radius) && dz <= int32(r.radius)
}

// ProtoChunk returns the ProtoChunk for the given chunk position, if present.
func (r *WorldGenRegion) ProtoChunk(p chunk.Pos) (*ProtoChunk, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ch, ok := r.chunks[p]
	return ch, ok
}

// chunkAt returns the ProtoChunk containing world coordinate (worldX, worldZ).
func (r *WorldGenRegion) chunkAt(worldX, worldZ int) (*ProtoChunk, error) {
	if !r.InRange(worldX, worldZ) {
		return nil, fmt.Errorf("%w: (%d, %d)", ErrOutOfRegion, worldX, worldZ)
	}
	p := chunk.Pos{int32(worldX >> 4), int32(worldZ >> 4)}
	r.mu.RLock()
	ch, ok := r.chunks[p]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: chunk %v", ErrMissingChunk, p)
	}
	return ch, nil
}

// Block returns the runtime block ID at absolute world coordinates (worldX, y, worldZ).
func (r *WorldGenRegion) Block(worldX, y, worldZ int) (uint32, error) {
	ch, err := r.chunkAt(worldX, worldZ)
	if err != nil {
		return 0, err
	}
	localX := uint8(worldX & 0xf)
	localZ := uint8(worldZ & 0xf)
	return ch.Block(localX, int16(y), localZ, 0), nil
}

// SetBlock mutates the runtime block ID at absolute world coordinates (worldX, y, worldZ).
func (r *WorldGenRegion) SetBlock(worldX, y, worldZ int, runtimeID uint32) error {
	ch, err := r.chunkAt(worldX, worldZ)
	if err != nil {
		return err
	}
	localX := uint8(worldX & 0xf)
	localZ := uint8(worldZ & 0xf)
	ch.SetBlock(localX, int16(y), localZ, 0, runtimeID)
	return nil
}

// Biome returns the biome ID at absolute world coordinates (worldX, y, worldZ).
func (r *WorldGenRegion) Biome(worldX, y, worldZ int) (uint32, error) {
	ch, err := r.chunkAt(worldX, worldZ)
	if err != nil {
		return 0, err
	}
	localX := uint8(worldX & 0xf)
	localZ := uint8(worldZ & 0xf)
	return ch.Biome(localX, int16(y), localZ), nil
}

// SetBiome updates the biome ID at absolute world coordinates (worldX, y, worldZ).
func (r *WorldGenRegion) SetBiome(worldX, y, worldZ int, biomeID uint32) error {
	ch, err := r.chunkAt(worldX, worldZ)
	if err != nil {
		return err
	}
	localX := uint8(worldX & 0xf)
	localZ := uint8(worldZ & 0xf)
	ch.SetBiome(localX, int16(y), localZ, biomeID)
	return nil
}

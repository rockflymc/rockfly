package generator

import (
	"fmt"
	"sync"

	"github.com/df-mc/dragonfly/server/world/chunk"
)

// GenerationStage represents the sequential pipeline stages a ProtoChunk transitions through.
type GenerationStage uint8

const (
	// StageEmpty indicates the chunk is newly allocated and empty.
	StageEmpty GenerationStage = iota
	// StageStructureStarts indicates global structures have registered anchor points in this chunk.
	StageStructureStarts
	// StageStructureReferences indicates cross-chunk bounding box references have been propagated.
	StageStructureReferences
	// StageBiomes indicates 3D Voronoi climate parameters and biomes have been calculated.
	StageBiomes
	// StageNoise indicates base terrain density sampling (stone, deepslate, water) is complete.
	StageNoise
	// StageSurface indicates biome surface rules (grass, sand, badlands strata) have been applied.
	StageSurface
	// StageCarvers indicates caves, ravines, and aquifers have carved underground space.
	StageCarvers
	// StageFeatures indicates ores, trees, vegetation, and minor features have been placed.
	StageFeatures
	// StageFull indicates the chunk generation is complete and ready for lighting and active gameplay.
	StageFull
)

// String returns a human-readable representation of GenerationStage.
func (s GenerationStage) String() string {
	switch s {
	case StageEmpty:
		return "Empty"
	case StageStructureStarts:
		return "StructureStarts"
	case StageStructureReferences:
		return "StructureReferences"
	case StageBiomes:
		return "Biomes"
	case StageNoise:
		return "Noise"
	case StageSurface:
		return "Surface"
	case StageCarvers:
		return "Carvers"
	case StageFeatures:
		return "Features"
	case StageFull:
		return "Full"
	default:
		return fmt.Sprintf("Stage(%d)", s)
	}
}

// ProtoChunk represents a chunk in the generation staging pipeline before it is committed
// to the world for active ticking. It wraps the raw *chunk.Chunk and tracks generation status
// and structure data with thread-safe mutation primitives.
type ProtoChunk struct {
	*chunk.Chunk
	mu         sync.RWMutex
	pos        chunk.Pos
	stage      GenerationStage
	structures map[string]any
}

// NewProtoChunk constructs a new ProtoChunk at the given position wrapping a raw *chunk.Chunk.
func NewProtoChunk(pos chunk.Pos, c *chunk.Chunk) *ProtoChunk {
	return &ProtoChunk{
		Chunk:      c,
		pos:        pos,
		stage:      StageEmpty,
		structures: make(map[string]any),
	}
}

// Pos returns the chunk position.
func (p *ProtoChunk) Pos() chunk.Pos {
	return p.pos
}

// Stage returns the current generation stage of the ProtoChunk.
func (p *ProtoChunk) Stage() GenerationStage {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stage
}

// SetStage advances or sets the generation stage.
func (p *ProtoChunk) SetStage(stage GenerationStage) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stage = stage
}

// SetStructureStart registers a structure start anchor in this chunk.
func (p *ProtoChunk) SetStructureStart(id string, data any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.structures[id] = data
}

// StructureStart retrieves a registered structure start anchor.
func (p *ProtoChunk) StructureStart(id string) (any, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	val, ok := p.structures[id]
	return val, ok
}

// RawChunk returns the underlying *chunk.Chunk.
func (p *ProtoChunk) RawChunk() *chunk.Chunk {
	return p.Chunk
}

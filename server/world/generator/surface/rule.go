package surface

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// ProtoChunk represents an un-ticked chunk column undergoing generation staging
// before being promoted into the ticking lifecycle.
type ProtoChunk struct {
	*chunk.Chunk
	Stage int
}

// NewProtoChunk wraps a *chunk.Chunk in a ProtoChunk container.
func NewProtoChunk(c *chunk.Chunk) *ProtoChunk {
	return &ProtoChunk{Chunk: c}
}

// SurfaceRule defines the interface for rule-based surface block placement.
type SurfaceRule interface {
	Apply(x, y, z int, depth int, biome world.Biome, chunk *ProtoChunk) bool
}

package surface

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator"
)

// ProtoChunk is an alias to generator.ProtoChunk.
type ProtoChunk = generator.ProtoChunk

// NewProtoChunk wraps a *chunk.Chunk in a ProtoChunk container.
func NewProtoChunk(c *chunk.Chunk) *ProtoChunk {
	return generator.NewProtoChunk(chunk.Pos{0, 0}, c)
}

// SurfaceRule defines the interface for rule-based surface block placement.
type SurfaceRule interface {
	Apply(x, y, z int, depth int, biome world.Biome, chunk *ProtoChunk) bool
}

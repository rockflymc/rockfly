package vanilla

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Piece represents a distinct sub-component of a structure with its own bounding box.
type Piece struct {
	Name string
	BBox BoundingBox
}

// Instance represents a placed vanilla structure instance in the world.
type Instance struct {
	Type   Type
	Origin cube.Pos
	BBox   BoundingBox
	Pieces []Piece
}

// Manager maintains thread-safe spatial indexing of active structures across chunks.
type Manager struct {
	mu      sync.RWMutex
	byChunk map[world.ChunkPos][]*Instance
	all     []*Instance
}

var (
	defaultManager     *Manager
	defaultManagerOnce sync.Once
)

// DefaultManager returns the global singleton structure manager.
func DefaultManager() *Manager {
	defaultManagerOnce.Do(func() {
		defaultManager = NewManager()
	})
	return defaultManager
}

// NewManager initializes an empty structure manager.
func NewManager() *Manager {
	return &Manager{
		byChunk: make(map[world.ChunkPos][]*Instance),
		all:     nil,
	}
}

// Add registers a structure instance and indexes all overlapping chunk columns.
func (m *Manager) Add(inst *Instance) {
	if m == nil || inst == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.all = append(m.all, inst)

	minChunkX := inst.BBox.Min.X() >> 4
	maxChunkX := inst.BBox.Max.X() >> 4
	minChunkZ := inst.BBox.Min.Z() >> 4
	maxChunkZ := inst.BBox.Max.Z() >> 4

	for cx := minChunkX; cx <= maxChunkX; cx++ {
		for cz := minChunkZ; cz <= maxChunkZ; cz++ {
			cPos := world.ChunkPos{int32(cx), int32(cz)}
			m.byChunk[cPos] = append(m.byChunk[cPos], inst)
		}
	}
}

// StructuresAt returns all structure instances overlapping the specified chunk.
func (m *Manager) StructuresAt(pos world.ChunkPos) []*Instance {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.byChunk[pos]
}

// InsideStructure reports whether world coordinates pos lie within any registered structure of type t.
func (m *Manager) InsideStructure(pos cube.Pos, t Type) bool {
	if m == nil {
		return false
	}
	cPos := world.ChunkPos{int32(pos.X() >> 4), int32(pos.Z() >> 4)}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, inst := range m.byChunk[cPos] {
		if inst.Type == t && inst.BBox.Contains(pos) {
			return true
		}
	}
	return false
}

// InsidePiece reports whether pos lies within a named piece sub-component of structure type t.
func (m *Manager) InsidePiece(pos cube.Pos, t Type, pieceName string) bool {
	if m == nil {
		return false
	}
	cPos := world.ChunkPos{int32(pos.X() >> 4), int32(pos.Z() >> 4)}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, inst := range m.byChunk[cPos] {
		if inst.Type == t && inst.BBox.Contains(pos) {
			for _, p := range inst.Pieces {
				if p.Name == pieceName && p.BBox.Contains(pos) {
					return true
				}
			}
		}
	}
	return false
}

// Clear flushes all indexed structure instances.
func (m *Manager) Clear() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.byChunk = make(map[world.ChunkPos][]*Instance)
	m.all = nil
}

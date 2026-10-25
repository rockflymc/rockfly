package world

import (
	"math"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

// CellShift is the bit shift representing the cell size in blocks (1 << 5 = 32 blocks).
const CellShift = 5

// CellSize is the length of one side of a spatial bucket in blocks (32).
const CellSize = 1 << CellShift

// MaxQuerySpan is the maximum number of bucket cells allowed along any axis in a spatial query.
const MaxQuerySpan = 64

// CellCoord represents a 2D bucket coordinate in the spatial hash grid.
type CellCoord struct {
	X, Z int32
}

// CellCoordFromVec3 returns the CellCoord corresponding to the 3D position vector.
func CellCoordFromVec3(pos mgl64.Vec3) CellCoord {
	x := pos[0]
	z := pos[2]
	if math.IsNaN(x) || math.IsInf(x, 0) {
		x = 0
	}
	if math.IsNaN(z) || math.IsInf(z, 0) {
		z = 0
	}
	return CellCoord{
		X: int32(int(math.Floor(x)) >> CellShift),
		Z: int32(int(math.Floor(z)) >> CellShift),
	}
}

// CellCoordFromXZ returns the CellCoord for block coordinates x and z.
func CellCoordFromXZ(x, z int) CellCoord {
	return CellCoord{
		X: int32(x >> CellShift),
		Z: int32(z >> CellShift),
	}
}

// SpatialGrid implements an O(1) constant-time 2D spatial hash bucket grid for entity tracking,
// visibility evaluation, and low-latency spatial queries.
type SpatialGrid struct {
	mu sync.RWMutex

	cells map[CellCoord][]*EntityHandle
	locs  map[*EntityHandle]CellCoord
}

// NewSpatialGrid creates an initialized, thread-safe spatial grid.
func NewSpatialGrid() *SpatialGrid {
	return &SpatialGrid{
		cells: make(map[CellCoord][]*EntityHandle),
		locs:  make(map[*EntityHandle]CellCoord),
	}
}

// initMaps guarantees that internal maps are allocated even if SpatialGrid is constructed without NewSpatialGrid.
func (g *SpatialGrid) initMaps() {
	if g.cells == nil {
		g.cells = make(map[CellCoord][]*EntityHandle)
	}
	if g.locs == nil {
		g.locs = make(map[*EntityHandle]CellCoord)
	}
}

// Insert registers an entity handle in the spatial grid at the specified position.
func (g *SpatialGrid) Insert(handle *EntityHandle, pos mgl64.Vec3) {
	if g == nil || handle == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	g.initMaps()
	cell := CellCoordFromVec3(pos)
	g.locs[handle] = cell
	g.cells[cell] = append(g.cells[cell], handle)
}

// Remove unregisters an entity handle from the spatial grid.
func (g *SpatialGrid) Remove(handle *EntityHandle) {
	if g == nil || handle == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.locs == nil || g.cells == nil {
		return
	}

	cell, ok := g.locs[handle]
	if !ok {
		return
	}
	delete(g.locs, handle)

	entries := g.cells[cell]
	for i, h := range entries {
		if h == handle {
			last := len(entries) - 1
			entries[i] = entries[last]
			entries[last] = nil
			entries = entries[:last]
			break
		}
	}
	if len(entries) == 0 {
		delete(g.cells, cell)
	} else {
		g.cells[cell] = entries
	}
}

// Move updates an entity handle's position in the spatial grid, atomically moving it
// across bucket cells if its cell coordinate changed.
func (g *SpatialGrid) Move(handle *EntityHandle, newPos mgl64.Vec3) bool {
	if g == nil || handle == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	g.initMaps()
	oldCell, ok := g.locs[handle]
	newCell := CellCoordFromVec3(newPos)
	if ok && oldCell == newCell {
		return false
	}

	if ok {
		entries := g.cells[oldCell]
		for i, h := range entries {
			if h == handle {
				last := len(entries) - 1
				entries[i] = entries[last]
				entries[last] = nil
				entries = entries[:last]
				break
			}
		}
		if len(entries) == 0 {
			delete(g.cells, oldCell)
		} else {
			g.cells[oldCell] = entries
		}
	}

	g.locs[handle] = newCell
	g.cells[newCell] = append(g.cells[newCell], handle)
	return true
}

// Query performs a zero-allocation spatial bounding box search across overlapping cells.
// The provided callback function f is invoked for each entity within the box until f returns false.
func (g *SpatialGrid) Query(box cube.BBox, f func(h *EntityHandle) bool) {
	if g == nil || f == nil {
		return
	}

	minCoord := CellCoordFromXZ(int(math.Floor(box.Min()[0])), int(math.Floor(box.Min()[2])))
	maxCoord := CellCoordFromXZ(int(math.Floor(box.Max()[0])), int(math.Floor(box.Max()[2])))

	if minCoord.X > maxCoord.X {
		minCoord.X, maxCoord.X = maxCoord.X, minCoord.X
	}
	if minCoord.Z > maxCoord.Z {
		minCoord.Z, maxCoord.Z = maxCoord.Z, minCoord.Z
	}
	if maxCoord.X-minCoord.X > MaxQuerySpan {
		maxCoord.X = minCoord.X + MaxQuerySpan
	}
	if maxCoord.Z-minCoord.Z > MaxQuerySpan {
		maxCoord.Z = minCoord.Z + MaxQuerySpan
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.cells == nil {
		return
	}

	for x := minCoord.X; x <= maxCoord.X; x++ {
		for z := minCoord.Z; z <= maxCoord.Z; z++ {
			handles, ok := g.cells[CellCoord{X: x, Z: z}]
			if !ok {
				continue
			}
			for _, h := range handles {
				if box.Vec3Within(h.data.Pos) {
					if !f(h) {
						return
					}
				}
			}
		}
	}
}

// QueryRadius performs a zero-allocation radial search around center within the given distance.
func (g *SpatialGrid) QueryRadius(center mgl64.Vec3, radius float64, f func(h *EntityHandle) bool) {
	if g == nil || f == nil {
		return
	}
	if math.IsNaN(radius) || math.IsInf(radius, 0) || radius <= 0 {
		return
	}
	if math.IsNaN(center[0]) || math.IsNaN(center[1]) || math.IsNaN(center[2]) {
		return
	}
	if math.IsInf(center[0], 0) || math.IsInf(center[1], 0) || math.IsInf(center[2], 0) {
		return
	}
	if radius > 512.0 {
		radius = 512.0
	}

	minCoord := CellCoordFromXZ(int(math.Floor(center[0]-radius)), int(math.Floor(center[2]-radius)))
	maxCoord := CellCoordFromXZ(int(math.Floor(center[0]+radius)), int(math.Floor(center[2]+radius)))
	if maxCoord.X-minCoord.X > MaxQuerySpan {
		maxCoord.X = minCoord.X + MaxQuerySpan
	}
	if maxCoord.Z-minCoord.Z > MaxQuerySpan {
		maxCoord.Z = minCoord.Z + MaxQuerySpan
	}

	radSq := radius * radius

	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.cells == nil {
		return
	}

	for x := minCoord.X; x <= maxCoord.X; x++ {
		for z := minCoord.Z; z <= maxCoord.Z; z++ {
			handles, ok := g.cells[CellCoord{X: x, Z: z}]
			if !ok {
				continue
			}
			for _, h := range handles {
				dx := h.data.Pos[0] - center[0]
				dz := h.data.Pos[2] - center[2]
				dy := h.data.Pos[1] - center[1]
				if dx*dx+dz*dz+dy*dy <= radSq {
					if !f(h) {
						return
					}
				}
			}
		}
	}
}

// CellEntities returns a slice of all entity handles in the given cell.
func (g *SpatialGrid) CellEntities(cell CellCoord) []*EntityHandle {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.cells == nil {
		return nil
	}
	handles, ok := g.cells[cell]
	if !ok || len(handles) == 0 {
		return nil
	}
	res := make([]*EntityHandle, len(handles))
	copy(res, handles)
	return res
}

// EntityCount returns the total number of entities tracked in the grid.
func (g *SpatialGrid) EntityCount() int {
	if g == nil {
		return 0
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.locs)
}

// CellCount returns the total number of active non-empty cells in the grid.
func (g *SpatialGrid) CellCount() int {
	if g == nil {
		return 0
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.cells)
}

// Clear removes all entities and buckets from the spatial grid.
func (g *SpatialGrid) Clear() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cells != nil {
		clear(g.cells)
	}
	if g.locs != nil {
		clear(g.locs)
	}
}

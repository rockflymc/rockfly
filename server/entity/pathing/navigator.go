package pathing

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

// PathRequest holds an asynchronous pathfinding task submitted from the game thread.
type PathRequest struct {
	Start, Target mgl64.Vec3
	Provider      BlockProvider
	Callback      func(path *Path)
}

// Navigator manages off-thread 3D A* path calculation across a pool of background workers,
// eliminating main-thread tick lag and stalling during complex entity navigation.
type Navigator struct {
	astar   *AStar
	queue   chan PathRequest
	workers int
	closed  atomic.Bool
	wg      sync.WaitGroup
}

// NewNavigator spawns a background worker pool dedicated to processing path requests.
// If workers <= 0, runtime.NumCPU() is used.
func NewNavigator(workers int, conf AStarConfig) *Navigator {
	if workers <= 0 {
		workers = max(2, runtime.NumCPU())
	}
	n := &Navigator{
		astar:   NewAStar(conf),
		queue:   make(chan PathRequest, 1024),
		workers: workers,
	}

	n.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go n.workerLoop()
	}
	return n
}

// Submit non-blockingly enqueues a pathfinding query to the background worker pool.
// Returns true if successfully queued, or false if the navigator is closed or queue is saturated.
func (n *Navigator) Submit(req PathRequest) bool {
	if n.closed.Load() {
		return false
	}
	select {
	case n.queue <- req:
		return true
	default:
		// Saturated queue: drop request to avoid stalling caller
		return false
	}
}

// FindPathSync computes a path synchronously for tests or urgent calculations.
func (n *Navigator) FindPathSync(start, target mgl64.Vec3, provider BlockProvider) *Path {
	startPos := cube.PosFromVec3(start)
	targetPos := cube.PosFromVec3(target)

	voxels := n.astar.FindPath(startPos, targetPos, provider)
	if len(voxels) == 0 {
		return nil
	}

	waypoints := make([]mgl64.Vec3, len(voxels))
	for i, v := range voxels {
		// Use center of block for smooth mob movement
		waypoints[i] = v.Vec3Centre()
	}
	return NewPath(waypoints)
}

func (n *Navigator) workerLoop() {
	defer n.wg.Done()

	for req := range n.queue {
		path := n.FindPathSync(req.Start, req.Target, req.Provider)
		if req.Callback != nil {
			req.Callback(path)
		}
	}
}

// Close gracefully terminates all background pathfinding workers.
func (n *Navigator) Close() error {
	if n.closed.CompareAndSwap(false, true) {
		close(n.queue)
		n.wg.Wait()
	}
	return nil
}

package pathing

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type mockWorld struct {
	mu     sync.RWMutex
	blocks map[cube.Pos]world.Block
}

func newMockWorld() *mockWorld {
	return &mockWorld{blocks: make(map[cube.Pos]world.Block)}
}

func (m *mockWorld) Block(pos cube.Pos) world.Block {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.blocks[pos]
	if !ok {
		return block.Air{}
	}
	return b
}

func (m *mockWorld) Set(pos cube.Pos, b world.Block) {
	m.mu.Lock()
	m.blocks[pos] = b
	m.mu.Unlock()
}

// buildFloor creates a solid stone floor at Y with air above it.
func (m *mockWorld) buildFloor(minX, minZ, maxX, maxZ, y int) {
	stone := block.Stone{}
	for x := minX; x <= maxX; x++ {
		for z := minZ; z <= maxZ; z++ {
			m.Set(cube.Pos{x, y, z}, stone)
		}
	}
}

func TestAStar_FlatStraightPath(t *testing.T) {
	w := newMockWorld()
	w.buildFloor(0, 0, 10, 10, 0) // Floor at Y=0, walking level at Y=1

	finder := NewAStar(DefaultAStarConfig())
	start := cube.Pos{1, 1, 1}
	target := cube.Pos{5, 1, 1}

	path := finder.FindPath(start, target, w)
	if len(path) == 0 {
		t.Fatalf("expected a path from %v to %v", start, target)
	}

	if path[len(path)-1] != target {
		t.Fatalf("expected end of path to reach target %v, got %v", target, path[len(path)-1])
	}
}

func TestAStar_ObstacleAvoidance(t *testing.T) {
	w := newMockWorld()
	w.buildFloor(0, 0, 10, 10, 0) // Floor at Y=0

	// Erect a wall at X=3 from Z=0 to Z=7
	for z := 0; z <= 7; z++ {
		w.Set(cube.Pos{3, 1, z}, block.Stone{})
		w.Set(cube.Pos{3, 2, z}, block.Stone{})
	}

	finder := NewAStar(DefaultAStarConfig())
	start := cube.Pos{1, 1, 3}
	target := cube.Pos{5, 1, 3}

	path := finder.FindPath(start, target, w)
	if len(path) == 0 {
		t.Fatalf("expected path around wall from %v to %v", start, target)
	}

	// Verify path avoids the wall blocks
	for _, p := range path {
		if p[0] == 3 && p[2] <= 7 {
			t.Fatalf("path routed through wall block: %v", p)
		}
	}

	if path[len(path)-1] != target {
		t.Fatalf("expected path to reach target %v, got %v", target, path[len(path)-1])
	}
}

func TestAStar_JumpAndFall(t *testing.T) {
	w := newMockWorld()

	// Low platform at X=1..3, Y=0 (feet at Y=1)
	w.buildFloor(1, 1, 3, 3, 0)

	// Elevated platform at X=4..6, Y=1 (feet at Y=2)
	w.buildFloor(4, 1, 6, 3, 1)

	// Drop down platform at X=7..9, Y=0 (feet at Y=1)
	w.buildFloor(7, 1, 9, 3, 0)

	finder := NewAStar(DefaultAStarConfig())
	start := cube.Pos{2, 1, 2}
	target := cube.Pos{8, 1, 2}

	path := finder.FindPath(start, target, w)
	if len(path) == 0 {
		t.Fatalf("expected path across elevation changes")
	}

	hasJump := false
	hasFall := false
	for _, p := range path {
		if p[1] == 2 {
			hasJump = true
		}
		if hasJump && p[1] == 1 {
			hasFall = true
		}
	}

	if !hasJump || !hasFall {
		t.Fatalf("expected path to jump up (Y=2) and step down (Y=1), got path: %v", path)
	}
}

func TestAStar_HazardAvoidance(t *testing.T) {
	w := newMockWorld()
	w.buildFloor(0, 0, 10, 10, 0)

	// Place lava pool directly in center (X=3..4, Z=1..2)
	for x := 3; x <= 4; x++ {
		for z := 1; z <= 2; z++ {
			w.Set(cube.Pos{x, 1, z}, block.Lava{Depth: 8, Still: true})
		}
	}

	finder := NewAStar(DefaultAStarConfig())
	start := cube.Pos{1, 1, 1}
	target := cube.Pos{6, 1, 1}

	path := finder.FindPath(start, target, w)
	if len(path) == 0 {
		t.Fatalf("expected path around lava pool")
	}

	// Verify no path node passes through the lava
	for _, p := range path {
		if (p[0] == 3 || p[0] == 4) && (p[2] == 1 || p[2] == 2) {
			t.Fatalf("path routed into lava: %v", p)
		}
	}
}

func TestAStar_DiagonalCornerClearance(t *testing.T) {
	w := newMockWorld()
	w.buildFloor(0, 0, 5, 5, 0)

	// Place solid wall corner at (2, 1, 1) and (1, 1, 2)
	w.Set(cube.Pos{2, 1, 1}, block.Stone{})
	w.Set(cube.Pos{2, 2, 1}, block.Stone{})
	w.Set(cube.Pos{1, 1, 2}, block.Stone{})
	w.Set(cube.Pos{1, 2, 2}, block.Stone{})

	finder := NewAStar(DefaultAStarConfig())
	start := cube.Pos{1, 1, 1}
	target := cube.Pos{2, 1, 2}

	path := finder.FindPath(start, target, w)
	// Direct diagonal from (1, 1, 1) to (2, 1, 2) should NOT be taken because both
	// cardinal neighbors (2, 1, 1) and (1, 1, 2) are solid walls!
	if len(path) > 0 {
		if path[0] == target && len(path) == 1 {
			t.Fatalf("path illegally cut through wall corner: %v", path)
		}
	}
}

func TestNavigator_AsyncDispatch(t *testing.T) {
	w := newMockWorld()
	w.buildFloor(0, 0, 20, 20, 0)

	nav := NewNavigator(4, DefaultAStarConfig())
	defer nav.Close()

	var wg sync.WaitGroup
	count := 50
	completed := make(chan bool, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		startX := float64(1 + (i % 5))
		targetX := float64(10 + (i % 5))

		req := PathRequest{
			Start:    mgl64.Vec3{startX, 1, 2},
			Target:   mgl64.Vec3{targetX, 1, 2},
			Provider: w,
			Callback: func(path *Path) {
				defer wg.Done()
				if path != nil && path.Length() > 0 {
					completed <- true
				}
			},
		}

		ok := nav.Submit(req)
		if !ok {
			t.Fatalf("failed to submit request %v", i)
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All 50 queries resolved asynchronously
		if len(completed) != count {
			t.Fatalf("expected %v successful async path calculations, got %v", count, len(completed))
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("async pathfinding timed out")
	}
}

func TestNavigator_NaNAndNilSafety(t *testing.T) {
	w := newMockWorld()
	nav := NewNavigator(2, DefaultAStarConfig())
	defer nav.Close()

	// Nil provider should not panic
	path := nav.FindPathSync(mgl64.Vec3{1, 1, 1}, mgl64.Vec3{5, 1, 1}, nil)
	if path != nil {
		t.Fatalf("expected nil path with nil provider")
	}

	// NaN coordinates should not panic or hang
	nan := math.NaN()
	path = nav.FindPathSync(mgl64.Vec3{nan, 1, 1}, mgl64.Vec3{5, 1, 1}, w)
	if path != nil {
		t.Fatalf("expected nil path with NaN coordinate")
	}

	// Nil navigator should not panic
	var nilNav *Navigator
	if nilNav.Submit(PathRequest{}) {
		t.Fatalf("nil navigator submit should return false")
	}
	if nilNav.FindPathSync(mgl64.Vec3{1, 1, 1}, mgl64.Vec3{5, 1, 1}, w) != nil {
		t.Fatalf("nil navigator FindPathSync should return nil")
	}
}

func BenchmarkAStar_Maze(b *testing.B) {
	w := newMockWorld()
	w.buildFloor(0, 0, 30, 30, 0)

	// Alternating hurdles
	for x := 5; x <= 25; x += 4 {
		for z := 2; z <= 26; z++ {
			if z%6 != 0 {
				w.Set(cube.Pos{x, 1, z}, block.Stone{})
				w.Set(cube.Pos{x, 2, z}, block.Stone{})
			}
		}
	}

	finder := NewAStar(DefaultAStarConfig())
	start := cube.Pos{2, 1, 2}
	target := cube.Pos{28, 1, 28}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = finder.FindPath(start, target, w)
	}
}

func BenchmarkNavigator_ConcurrentQueries(b *testing.B) {
	w := newMockWorld()
	w.buildFloor(0, 0, 30, 30, 0)

	nav := NewNavigator(8, DefaultAStarConfig())
	defer nav.Close()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = nav.FindPathSync(mgl64.Vec3{2, 1, 2}, mgl64.Vec3{20, 1, 20}, w)
		}
	})
}

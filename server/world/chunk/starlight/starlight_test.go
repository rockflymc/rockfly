package starlight

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestPackedNode_EncodeDecode(t *testing.T) {
	testCases := []struct {
		x, y, z int
		level   uint8
		isSky   bool
	}{
		{0, 0, 0, 15, false},
		{15, 64, 15, 14, true},
		{47, -64, 47, 0, false},
		{63, 319, 63, 1, true},
		{32, 128, 16, 7, false},
		{5, -1, 10, 12, true},
	}

	for _, tc := range testCases {
		packed := PackNode(tc.x, tc.y, tc.z, tc.level, tc.isSky)

		if packed.X() != tc.x {
			t.Errorf("expected X %d, got %d", tc.x, packed.X())
		}
		if packed.Y() != tc.y {
			t.Errorf("expected Y %d, got %d", tc.y, packed.Y())
		}
		if packed.Z() != tc.z {
			t.Errorf("expected Z %d, got %d", tc.z, packed.Z())
		}
		if packed.Level() != tc.level {
			t.Errorf("expected Level %d, got %d", tc.level, packed.Level())
		}
		if packed.IsSky() != tc.isSky {
			t.Errorf("expected IsSky %v, got %v", tc.isSky, packed.IsSky())
		}

		baseX, baseZ := 160, 320
		expectedPos := cube.Pos{baseX + tc.x, tc.y, baseZ + tc.z}
		if pos := packed.Pos(baseX, baseZ); pos != expectedPos {
			t.Errorf("expected Pos %v, got %v", expectedPos, pos)
		}
	}
}

func TestPackedQueue_FIFO(t *testing.T) {
	q := NewPackedQueue(4)

	if !q.Empty() {
		t.Fatal("expected empty queue")
	}

	// Push 10 elements to force ring buffer growth
	for i := 0; i < 10; i++ {
		q.Push(PackNode(i, i*10, i, uint8(i%16), i%2 == 0))
	}

	if q.Len() != 10 {
		t.Fatalf("expected len 10, got %d", q.Len())
	}

	for i := 0; i < 10; i++ {
		n, ok := q.Pop()
		if !ok {
			t.Fatalf("failed to pop element %d", i)
		}
		if n.X() != i || n.Y() != i*10 || n.Z() != i || n.Level() != uint8(i%16) {
			t.Fatalf("element %d mismatch: X=%d Y=%d Z=%d Level=%d", i, n.X(), n.Y(), n.Z(), n.Level())
		}
	}

	if !q.Empty() {
		t.Fatal("expected queue to be empty after popping all elements")
	}
}

func TestSliceMask_BitOps(t *testing.T) {
	var m SliceMask

	if !m.Empty() {
		t.Fatal("expected empty mask initially")
	}
	if m.Full() {
		t.Fatal("expected non-full mask initially")
	}

	// Set diagonals
	for i := 0; i < 16; i++ {
		m.Set(i, i)
	}

	if m.Empty() {
		t.Fatal("expected non-empty mask after sets")
	}
	if cnt := m.Count(); cnt != 16 {
		t.Fatalf("expected count 16, got %d", cnt)
	}

	for i := 0; i < 16; i++ {
		if !m.Get(i, i) {
			t.Fatalf("expected (i, i) to be set for %d", i)
		}
		if i > 0 && m.Get(i, i-1) {
			t.Fatalf("expected (i, i-1) to NOT be set for %d", i)
		}
	}

	// Invert mask
	inv := m.Not()
	if inv.Count() != 256-16 {
		t.Fatalf("expected inverted count %d, got %d", 256-16, inv.Count())
	}

	// AND with self should equal self
	andSelf := m.And(m)
	if andSelf != m {
		t.Fatal("expected m AND m == m")
	}

	// OR with inversion should be full
	orInv := m.Or(inv)
	if !orInv.Full() {
		t.Fatal("expected m OR inv to be Full")
	}
}

func TestEngine_AsyncExecution(t *testing.T) {
	engine := NewEngine(4)
	defer engine.Close()

	const tasksCount = 50
	var completed atomic.Int32
	var wg sync.WaitGroup
	wg.Add(tasksCount)

	for i := 0; i < tasksCount; i++ {
		taskIndex := i
		engine.Submit(LightTask{
			Type: TaskUpdate,
			Action: func() {
				// Simulate lighting flood-fill calculation
				time.Sleep(time.Duration(taskIndex%5) * time.Millisecond)
				completed.Add(1)
			},
			OnDone: func() {
				wg.Done()
			},
		})
	}

	wg.Wait()

	if val := completed.Load(); val != tasksCount {
		t.Fatalf("expected %d completed tasks, got %d", tasksCount, val)
	}
}

func BenchmarkPackedNode_PackUnpack(b *testing.B) {
	b.ReportAllocs()
	var n PackedNode
	for i := 0; i < b.N; i++ {
		n = PackNode(i&0x3f, (i%300)-64, (i>>6)&0x3f, uint8(i&0x0f), (i&1) == 0)
		_ = n.X() + n.Y() + n.Z() + int(n.Level())
	}
}

func BenchmarkSliceMask_Raycast(b *testing.B) {
	var m SliceMask
	for i := 0; i < 16; i++ {
		m.Set(i, 8)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !m.Empty() {
			_ = m.Count()
		}
	}
}

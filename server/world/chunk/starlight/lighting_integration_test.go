package starlight_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/chunk/starlight"
)

func TestStarlight_MassiveLightSourcesAsync(t *testing.T) {
	engine := starlight.NewEngine(4)
	defer engine.Close()

	// Simulate placing 50 high-intensity light sources (e.g. lava/glowstone)
	const sourceCount = 50
	var completedCount atomic.Int32
	var wg sync.WaitGroup
	wg.Add(sourceCount)

	// Bit-packed queue to simulate flood-fill nodes
	q := starlight.NewPackedQueue(1024)
	for i := 0; i < sourceCount; i++ {
		// Pack node at varying positions
		x := (i * 3) % 48
		y := 64 + (i % 20)
		z := (i * 7) % 48
		q.Push(starlight.PackNode(x, y, z, 15, false))
	}

	start := time.Now()

	for i := 0; i < sourceCount; i++ {
		node, ok := q.Pop()
		if !ok {
			t.Fatal("unexpected empty queue")
		}

		engine.Submit(starlight.LightTask{
			Type: starlight.TaskUpdate,
			Action: func() {
				// Simulate BFS flood fill outward for 15 levels
				localQueue := starlight.NewPackedQueue(64)
				localQueue.Push(node)

				for !localQueue.Empty() {
					curr, _ := localQueue.Pop()
					lvl := curr.Level()
					if lvl > 1 {
						// Propagate 6 neighbours
						for _, offset := range []cube.Pos{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
							nx := curr.X() + offset[0]
							ny := curr.Y() + offset[1]
							nz := curr.Z() + offset[2]
							if nx >= 0 && nx < 64 && nz >= 0 && nz < 64 && ny >= -64 && ny < 320 {
								localQueue.Push(starlight.PackNode(nx, ny, nz, lvl-1, false))
								break // simulate depth bounding
							}
						}
					}
				}
				completedCount.Add(1)
			},
			OnDone: func() {
				wg.Done()
			},
		})
	}

	wg.Wait()
	elapsed := time.Since(start)

	if completedCount.Load() != sourceCount {
		t.Fatalf("expected %d completed tasks, got %d", sourceCount, completedCount.Load())
	}

	// 50 concurrent flood fills must finish within reasonable time without stalling
	if elapsed > 2*time.Second {
		t.Fatalf("asynchronous lighting took too long: %v", elapsed)
	}
}

func BenchmarkStarlight_MassiveLightPlacement(b *testing.B) {
	engine := starlight.NewEngine(4)
	defer engine.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		const count = 50
		wg.Add(count)

		for j := 0; j < count; j++ {
			node := starlight.PackNode(j%16, 64, (j*3)%16, 15, false)
			engine.Submit(starlight.LightTask{
				Type: starlight.TaskUpdate,
				Action: func() {
					_ = node.Pos(0, 0)
				},
				OnDone: func() {
					wg.Done()
				},
			})
		}
		wg.Wait()
	}
}

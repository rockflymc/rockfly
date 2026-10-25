package block

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func TestHopperOptimizer_SleepCycle(t *testing.T) {
	w := newTestWorld()
	defer w.Close()

	pos := cube.Pos{0, 64, 0}
	hopper := NewHopper()
	optimizer := HopperOptimizer{}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, hopper, nil)

		// First tick: idle with empty inventory -> should trigger sleep
		hopper.Tick(0, pos, tx)

		if !optimizer.ShouldSleep(pos, 1) {
			t.Fatal("expected hopper to enter sleep cycle after idle tick")
		}

		// At tick 4 (still within 8-tick sleep duration), hopper should sleep
		if !optimizer.ShouldSleep(pos, 4) {
			t.Fatal("expected hopper to remain asleep at tick 4")
		}

		// At tick 9 (past 8-tick sleep duration), hopper should wake
		if optimizer.ShouldSleep(pos, 9) {
			t.Fatal("expected hopper to wake up after sleep duration elapsed")
		}
	}).Wait(context.Background())
}

func TestHopperOptimizer_SuctionOcclusion(t *testing.T) {
	w := newTestWorld()
	defer w.Close()

	pos := cube.Pos{10, 64, 10}
	abovePos := pos.Side(cube.FaceUp)
	hopper := NewHopper()
	optimizer := HopperOptimizer{}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, hopper, nil)

		// Block above is solid obsidian
		tx.SetBlock(abovePos, Obsidian{}, nil)

		occluded := optimizer.CheckSuctionOcclusion(pos, tx)
		if !occluded {
			t.Fatal("expected suction to be occluded by solid obsidian above")
		}

		st := getHopperState(pos)
		if !st.suctionBlocked {
			t.Fatal("expected suctionBlocked state to be true")
		}

		// Remove obsidian, replace with air
		tx.SetBlock(abovePos, Air{}, nil)
		occluded = optimizer.CheckSuctionOcclusion(pos, tx)
		if occluded {
			t.Fatal("expected suction to NOT be occluded when air is above")
		}
		if st.suctionBlocked {
			t.Fatal("expected suctionBlocked state to be false with air above")
		}
	}).Wait(context.Background())
}

func TestHopperOptimizer_WakeOnUpdate(t *testing.T) {
	pos := cube.Pos{20, 64, 20}
	optimizer := HopperOptimizer{}

	optimizer.PutToSleep(pos, 0, 100)
	if !optimizer.ShouldSleep(pos, 10) {
		t.Fatal("expected hopper to sleep")
	}

	WakeHopper(pos)
	if optimizer.ShouldSleep(pos, 10) {
		t.Fatal("expected hopper to be awake after WakeHopper")
	}
}

func TestHopperOptimizer_EmptyAndFullTargetSleep(t *testing.T) {
	optimizer := HopperOptimizer{}
	emptyPos := cube.Pos{30, 64, 30}
	fullPos := cube.Pos{40, 64, 40}

	// 1. Empty/idle hopper sleep (8 ticks)
	optimizer.PutToSleepEmpty(emptyPos, 0)
	if !optimizer.ShouldSleep(emptyPos, 4) {
		t.Fatal("expected empty hopper to sleep at tick 4")
	}
	if optimizer.ShouldSleep(emptyPos, 9) {
		t.Fatal("expected empty hopper to wake after 8 ticks (at tick 9)")
	}

	// 2. Full-target container sleep (80 ticks)
	optimizer.PutToSleepFullTarget(fullPos, 0)
	if !optimizer.ShouldSleep(fullPos, 40) {
		t.Fatal("expected full-target hopper to sleep at tick 40")
	}
	if !optimizer.ShouldSleep(fullPos, 79) {
		t.Fatal("expected full-target hopper to still sleep at tick 79")
	}
	if optimizer.ShouldSleep(fullPos, 81) {
		t.Fatal("expected full-target hopper to wake after 80 ticks (at tick 81)")
	}
}

func TestHopperOptimizer_NilAndBoundarySafety(t *testing.T) {
	optimizer := HopperOptimizer{}
	pos := cube.Pos{0, 64, 0}

	// 1. Nil Tx safety
	if optimizer.CheckSuctionOcclusion(pos, nil) {
		t.Fatal("expected false on nil Tx CheckSuctionOcclusion")
	}
	if optimizer.CollectDroppedItems(Hopper{}, pos, nil) {
		t.Fatal("expected false on nil Tx CollectDroppedItems")
	}

	// 2. Uninitialized Hopper (nil inventory) safety
	w := newTestWorld()
	defer w.Close()
	w.Do(func(tx *world.Tx) {
		if optimizer.CollectDroppedItems(Hopper{}, pos, tx) {
			t.Fatal("expected false on uninitialized Hopper CollectDroppedItems")
		}
	}).Wait(context.Background())
}

func BenchmarkHopperOptimizer_5000IdleHoppers(b *testing.B) {
	optimizer := HopperOptimizer{}
	positions := make([]cube.Pos, 5000)
	for i := 0; i < 5000; i++ {
		p := cube.Pos{i % 100, 64, i / 100}
		positions[i] = p
		optimizer.PutToSleep(p, 0, 1000)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var sleepingCount int
		for j := 0; j < 5000; j++ {
			if optimizer.ShouldSleep(positions[j], 10) {
				sleepingCount++
			}
		}
		if sleepingCount != 5000 {
			b.Fatalf("expected all 5000 hoppers to sleep, got %d", sleepingCount)
		}
	}
}

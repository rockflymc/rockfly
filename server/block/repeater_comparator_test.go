package block_test

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
)

// runTx executes f synchronously within a world transaction.
func runTx(w *world.World, f func(tx *world.Tx)) {
	w.Do(f).Wait(context.Background())
}

// TestRepeater_Delays verifies delay cycling and Bedrock block state encoding.
func TestRepeater_Delays(t *testing.T) {
	r := block.Repeater{Facing: cube.North, Delay: 1}

	for expected := 1; expected <= 4; expected++ {
		if r.Delay != expected {
			t.Fatalf("expected delay %d, got %d", expected, r.Delay)
		}
		name, props := r.EncodeBlock()
		if name != "minecraft:unpowered_repeater" {
			t.Fatalf("expected unpowered repeater name, got %s", name)
		}
		if delayProp, ok := props["repeater_delay"].(int32); !ok || int(delayProp) != expected-1 {
			t.Fatalf("expected repeater_delay %d, got %v", expected-1, props["repeater_delay"])
		}
		r.Delay = (r.Delay % 4) + 1
	}
}

// TestRepeater_DiodeOneWay verifies strict unidirectional power emission.
func TestRepeater_DiodeOneWay(t *testing.T) {
	r := block.Repeater{Facing: cube.North, Delay: 1, Powered: true}

	// Front face (North) should emit full 15 power and strong power.
	if p := r.RedstonePower(cube.Pos{}, nil, cube.FaceNorth); p != 15 {
		t.Fatalf("expected 15 power at front face, got %d", p)
	}
	if sp := r.RedstoneStrongPower(cube.Pos{}, nil, cube.FaceNorth); sp != 15 {
		t.Fatalf("expected 15 strong power at front face, got %d", sp)
	}

	// Rear (South), Left (West), Right (East), Top (Up), Bottom (Down) should emit 0.
	for _, face := range []cube.Face{cube.FaceSouth, cube.FaceWest, cube.FaceEast, cube.FaceUp, cube.FaceDown} {
		if p := r.RedstonePower(cube.Pos{}, nil, face); p != 0 {
			t.Fatalf("expected 0 power at face %v, got %d", face, p)
		}
		if sp := r.RedstoneStrongPower(cube.Pos{}, nil, face); sp != 0 {
			t.Fatalf("expected 0 strong power at face %v, got %d", face, sp)
		}
	}

	// When unpowered, front face should emit 0.
	r.Powered = false
	if p := r.RedstonePower(cube.Pos{}, nil, cube.FaceNorth); p != 0 {
		t.Fatalf("expected 0 power when unpowered, got %d", p)
	}
}

// TestRepeater_SideLocking verifies side-locking latches by another repeater and comparator.
func TestRepeater_SideLocking(t *testing.T) {
	w := world.Config{Dim: world.Overworld, Synchronous: true}.New()
	defer w.Close()

	pos := cube.Pos{0, 60, 0}
	loader := world.NewLoader(1, w, world.NopViewer{})
	runTx(w, func(tx *world.Tx) {
		loader.Move(tx, pos.Vec3())
	})
	defer runTx(w, func(tx *world.Tx) {
		loader.Close(tx)
	})

	runTx(w, func(tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceDown), block.Stone{}, nil)
		tx.SetBlock(pos.Side(cube.FaceWest).Side(cube.FaceDown), block.Stone{}, nil)
		tx.SetBlock(pos.Side(cube.FaceEast).Side(cube.FaceDown), block.Stone{}, nil)

		// Repeater A facing North at (0, 60, 0).
		repA := block.Repeater{Facing: cube.North, Delay: 1, Powered: true}
		tx.SetBlock(pos, repA, nil)

		if repA.SideLocked(pos, tx) {
			t.Fatalf("repeater should not be locked initially")
		}

		// Place Repeater B to the West (pos.Side(cube.FaceWest)), facing East (towards A), but unpowered.
		repB := block.Repeater{Facing: cube.East, Delay: 1, Powered: false}
		tx.SetBlock(pos.Side(cube.FaceWest), repB, nil)
		if repA.SideLocked(pos, tx) {
			t.Fatalf("unpowered side repeater should not lock repeater A")
		}

		// Power Repeater B.
		repB.Powered = true
		tx.SetBlock(pos.Side(cube.FaceWest), repB, nil)
		if !repA.SideLocked(pos, tx) {
			t.Fatalf("powered side repeater should lock repeater A")
		}

		// Remove Repeater B and place a powered Comparator to the East, facing West (towards A).
		tx.SetBlock(pos.Side(cube.FaceWest), block.Air{}, nil)
		comp := block.Comparator{Facing: cube.West, Powered: true, OutputSignal: 10}
		tx.SetBlock(pos.Side(cube.FaceEast), comp, nil)
		if !repA.SideLocked(pos, tx) {
			t.Fatalf("powered side comparator should lock repeater A")
		}

		// Unpower Comparator.
		comp.Powered = false
		comp.OutputSignal = 0
		tx.SetBlock(pos.Side(cube.FaceEast), comp, nil)
		if repA.SideLocked(pos, tx) {
			t.Fatalf("unpowered side comparator should not lock repeater A")
		}
	})
}

// TestComparator_ComparisonMode verifies comparison mode logic (rear >= side).
func TestComparator_ComparisonMode(t *testing.T) {
	w := world.Config{Dim: world.Overworld, Synchronous: true}.New()
	defer w.Close()

	pos := cube.Pos{10, 60, 10}
	loader := world.NewLoader(1, w, world.NopViewer{})
	runTx(w, func(tx *world.Tx) {
		loader.Move(tx, pos.Vec3())
	})
	defer runTx(w, func(tx *world.Tx) {
		loader.Close(tx)
	})

	runTx(w, func(tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceDown), block.Stone{}, nil)
		tx.SetBlock(pos.Side(cube.FaceSouth).Side(cube.FaceDown), block.Stone{}, nil)
		tx.SetBlock(pos.Side(cube.FaceWest).Side(cube.FaceDown), block.Stone{}, nil)

		// Comparator facing North, in comparison mode.
		c := block.Comparator{Facing: cube.North, Mode: block.ComparatorComparison}
		tx.SetBlock(pos, c, nil)

		// Provide rear input via a powered repeater pointing North at South side.
		rearRep := block.Repeater{Facing: cube.North, Delay: 1, Powered: true}
		tx.SetBlock(pos.Side(cube.FaceSouth), rearRep, nil)

		// Rear signal is 15. With side 0, output should be 15.
		c.ScheduledTick(pos, tx, nil)
		updated, ok := tx.Block(pos).(block.Comparator)
		if !ok || !updated.Powered || updated.OutputSignal != 15 {
			t.Fatalf("expected powered comparator with power 15, got %#v", updated)
		}

		// Now add equal side signal (15). Since 15 >= 15, output is still 15.
		sideRep := block.Repeater{Facing: cube.East, Delay: 1, Powered: true}
		tx.SetBlock(pos.Side(cube.FaceWest), sideRep, nil)
		c.ScheduledTick(pos, tx, nil)
		updated, _ = tx.Block(pos).(block.Comparator)
		if !updated.Powered || updated.OutputSignal != 15 {
			t.Fatalf("expected power 15 when rear == side, got %#v", updated)
		}

		// Now make side stronger than rear: rear comp with power 8, side has 15.
		rearComp := block.Comparator{Facing: cube.North, Powered: true, OutputSignal: 8}
		tx.SetBlock(pos.Side(cube.FaceSouth), rearComp, nil)
		c.ScheduledTick(pos, tx, nil)
		updated, _ = tx.Block(pos).(block.Comparator)
		if updated.Powered || updated.OutputSignal != 0 {
			t.Fatalf("expected power 0 when rear (8) < side (15), got %#v", updated)
		}
	})
}

// TestComparator_SubtractionMode verifies subtraction mode logic: max(0, rear - maxSide).
func TestComparator_SubtractionMode(t *testing.T) {
	w := world.Config{Dim: world.Overworld, Synchronous: true}.New()
	defer w.Close()

	pos := cube.Pos{20, 60, 20}
	loader := world.NewLoader(1, w, world.NopViewer{})
	runTx(w, func(tx *world.Tx) {
		loader.Move(tx, pos.Vec3())
	})
	defer runTx(w, func(tx *world.Tx) {
		loader.Close(tx)
	})

	runTx(w, func(tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceDown), block.Stone{}, nil)
		tx.SetBlock(pos.Side(cube.FaceSouth).Side(cube.FaceDown), block.Stone{}, nil)
		tx.SetBlock(pos.Side(cube.FaceWest).Side(cube.FaceDown), block.Stone{}, nil)

		// Comparator facing North, in Subtraction mode.
		c := block.Comparator{Facing: cube.North, Mode: block.ComparatorSubtraction}
		tx.SetBlock(pos, c, nil)

		// Rear component emits 15. Side component emits 5.
		rearComp := block.Comparator{Facing: cube.North, Powered: true, OutputSignal: 15}
		sideComp := block.Comparator{Facing: cube.East, Powered: true, OutputSignal: 5}
		tx.SetBlock(pos.Side(cube.FaceSouth), rearComp, nil)
		tx.SetBlock(pos.Side(cube.FaceWest), sideComp, nil)

		c.ScheduledTick(pos, tx, nil)
		updated, ok := tx.Block(pos).(block.Comparator)
		if !ok || !updated.Powered || updated.OutputSignal != 10 {
			t.Fatalf("expected power 10 (15 - 5), got %#v", updated)
		}

		// Now increase side component to 15. Subtraction: 15 - 15 = 0.
		sideComp.OutputSignal = 15
		tx.SetBlock(pos.Side(cube.FaceWest), sideComp, nil)
		c.ScheduledTick(pos, tx, nil)
		updated, _ = tx.Block(pos).(block.Comparator)
		if updated.Powered || updated.OutputSignal != 0 {
			t.Fatalf("expected power 0 (15 - 15), got %#v", updated)
		}

		// Test right-click toggles mode back to Comparison mode.
		c.Activate(pos, cube.FaceUp, tx, nil, nil)
		toggled, _ := tx.Block(pos).(block.Comparator)
		if toggled.Mode != block.ComparatorComparison {
			t.Fatalf("expected mode toggled to Comparison, got %v", toggled.Mode)
		}
	})
}

// TestComparator_ContainerReading verifies fullness calculation on inventory containers.
func TestComparator_ContainerReading(t *testing.T) {
	inv := inventory.New(27, nil)

	// Empty container -> 0.
	if sig := block.CalculateContainerSignal(inv); sig != 0 {
		t.Fatalf("expected signal 0 for empty container, got %d", sig)
	}

	// 1 item (stack of 1 cobblestone, max 64) -> signal 1.
	_ = inv.SetItem(0, item.NewStack(block.Cobblestone{}, 1))
	if sig := block.CalculateContainerSignal(inv); sig != 1 {
		t.Fatalf("expected signal 1 for 1 item, got %d", sig)
	}

	// Exactly half full: 27 stacks of 32 items -> signal 8.
	inv.Clear()
	for i := 0; i < 27; i++ {
		_ = inv.SetItem(i, item.NewStack(block.Cobblestone{}, 32))
	}
	if sig := block.CalculateContainerSignal(inv); sig != 8 {
		t.Fatalf("expected signal 8 for half-full container, got %d", sig)
	}

	// Completely full: 27 stacks of 64 items -> signal 15.
	inv.Clear()
	for i := 0; i < 27; i++ {
		_ = inv.SetItem(i, item.NewStack(block.Cobblestone{}, 64))
	}
	if sig := block.CalculateContainerSignal(inv); sig != 15 {
		t.Fatalf("expected signal 15 for completely full container, got %d", sig)
	}
}

// TestRepeaterComparator_NilAndBoundarySafety verifies defensive nil handling and bounds clamping.
func TestRepeaterComparator_NilAndBoundarySafety(t *testing.T) {
	// 1. Container signal nil safety
	if sig := block.CalculateContainerSignal(nil); sig != 0 {
		t.Fatalf("expected 0 for nil container, got %d", sig)
	}

	// 2. Container signal with overfilled / corrupted items
	inv := inventory.New(1, nil)
	_ = inv.SetItem(0, item.NewStack(block.Cobblestone{}, 1000))
	if sig := block.CalculateContainerSignal(inv); sig != 15 {
		t.Fatalf("expected clamped signal 15 for overfilled container, got %d", sig)
	}

	// 3. Repeater nil Tx safety
	rep := block.Repeater{Facing: cube.North, Delay: 1}
	if rep.Activate(cube.Pos{}, cube.FaceUp, nil, nil, nil) {
		t.Fatalf("expected false on nil Tx in Activate")
	}
	rep.NeighbourUpdateTick(cube.Pos{}, cube.Pos{}, nil)
	rep.ScheduledTick(cube.Pos{}, nil, nil)
	if rep.SideLocked(cube.Pos{}, nil) {
		t.Fatalf("expected false on nil Tx in SideLocked")
	}

	// 4. Comparator nil Tx safety
	comp := block.Comparator{Facing: cube.North}
	if comp.Activate(cube.Pos{}, cube.FaceUp, nil, nil, nil) {
		t.Fatalf("expected false on nil Tx in Activate")
	}
	comp.NeighbourUpdateTick(cube.Pos{}, cube.Pos{}, nil)
	comp.ScheduledTick(cube.Pos{}, nil, nil)

	// 5. Comparator NBT decoding clamps
	decoded := comp.DecodeNBT(map[string]any{"OutputSignal": int32(50)}).(block.Comparator)
	if decoded.OutputSignal != 15 || !decoded.Powered {
		t.Fatalf("expected clamped signal 15 and powered, got %d, %v", decoded.OutputSignal, decoded.Powered)
	}
	decodedNeg := comp.DecodeNBT(map[string]any{"OutputSignal": int32(-10)}).(block.Comparator)
	if decodedNeg.OutputSignal != 0 || decodedNeg.Powered {
		t.Fatalf("expected clamped signal 0 and unpowered, got %d, %v", decodedNeg.OutputSignal, decodedNeg.Powered)
	}
}

// BenchmarkRepeater_DiodeEvaluation measures repeater diode evaluation performance.
func BenchmarkRepeater_DiodeEvaluation(b *testing.B) {
	r := block.Repeater{Facing: cube.North, Delay: 1, Powered: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.RedstonePower(cube.Pos{}, nil, cube.FaceNorth)
	}
}

// BenchmarkComparator_ContainerSignal measures container signal calculation performance.
func BenchmarkComparator_ContainerSignal(b *testing.B) {
	inv := inventory.New(27, nil)
	for i := 0; i < 27; i++ {
		_ = inv.SetItem(i, item.NewStack(block.Cobblestone{}, 32))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = block.CalculateContainerSignal(inv)
	}
}

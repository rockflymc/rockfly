package block

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
)

func TestAutomation_EncodeDecode(t *testing.T) {
	// 1. Observer: 12 permutations (6 faces x 2 powered states)
	for _, o := range allObservers() {
		name, properties := o.EncodeBlock()
		if name != "minecraft:observer" {
			t.Fatalf("expected minecraft:observer, got %s", name)
		}
		faceDir, ok := properties["minecraft:facing_direction"].(string)
		if !ok || faceDir == "" {
			t.Fatalf("invalid facing_direction: %v", properties["minecraft:facing_direction"])
		}
		powered, ok := properties["powered_bit"].(uint8)
		if !ok || (powered != 0 && powered != 1) {
			t.Fatalf("invalid powered_bit: %v", properties["powered_bit"])
		}
	}

	// 2. Dispenser: 12 permutations (6 faces x 2 triggered states)
	for _, d := range allDispensers() {
		name, properties := d.EncodeBlock()
		if name != "minecraft:dispenser" {
			t.Fatalf("expected minecraft:dispenser, got %s", name)
		}
		facing, ok := properties["facing_direction"].(int32)
		if !ok || facing < 0 || facing > 5 {
			t.Fatalf("invalid facing_direction: %v", properties["facing_direction"])
		}
	}

	// 3. Dropper: 12 permutations (6 faces x 2 triggered states)
	for _, d := range allDroppers() {
		name, properties := d.EncodeBlock()
		if name != "minecraft:dropper" {
			t.Fatalf("expected minecraft:dropper, got %s", name)
		}
		facing, ok := properties["facing_direction"].(int32)
		if !ok || facing < 0 || facing > 5 {
			t.Fatalf("invalid facing_direction: %v", properties["facing_direction"])
		}
	}

	// 4. Crafter: 48 permutations (12 orientations x 2 crafting x 2 triggered)
	for _, c := range allCrafters() {
		name, properties := c.EncodeBlock()
		if name != "minecraft:crafter" {
			t.Fatalf("expected minecraft:crafter, got %s", name)
		}
		orient, ok := properties["orientation"].(string)
		if !ok || orient == "" {
			t.Fatalf("invalid orientation: %v", properties["orientation"])
		}
	}
}

func TestObserver_PulseOnUpdate(t *testing.T) {
	w := newTestWorld()
	observerPos := cube.Pos{0, 64, 0}
	obs := Observer{Facing: cube.FaceNorth, Powered: false}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(observerPos, obs, nil)

		// Rear face is FaceSouth. Initially unpowered (0).
		if p := obs.RedstonePower(observerPos, tx, cube.FaceSouth); p != 0 {
			t.Fatalf("expected initial unpowered 0, got %d", p)
		}

		// Non-rear faces never output power.
		if p := obs.RedstonePower(observerPos, tx, cube.FaceNorth); p != 0 {
			t.Fatalf("expected non-rear face power 0, got %d", p)
		}

		// Trigger block update at front (North)
		frontPos := observerPos.Side(cube.FaceNorth)
		obs.NeighbourUpdateTick(observerPos, frontPos, tx)

		// Scheduled tick 1: turns on
		obs.ScheduledTick(observerPos, tx, nil)
		currentObs, _ := tx.Block(observerPos).(Observer)
		if !currentObs.Powered {
			t.Fatal("expected observer to turn powered on tick 1")
		}
		if p := currentObs.RedstonePower(observerPos, tx, cube.FaceSouth); p != 15 {
			t.Fatalf("expected strong power 15 on rear face, got %d", p)
		}

		// Scheduled tick 2: pulse concludes, turns off
		currentObs.ScheduledTick(observerPos, tx, nil)
		finalObs, _ := tx.Block(observerPos).(Observer)
		if finalObs.Powered {
			t.Fatal("expected observer to turn off on tick 2")
		}
		if p := finalObs.RedstonePower(observerPos, tx, cube.FaceSouth); p != 0 {
			t.Fatalf("expected power 0 after pulse, got %d", p)
		}
	}).Wait(context.Background())
}

func TestDropper_DropAndTransfer(t *testing.T) {
	w := newTestWorld()
	dropperPos := cube.Pos{0, 64, 0}
	dropper := NewDropper()
	dropper.Facing = cube.FaceEast

	w.Do(func(tx *world.Tx) {
		// Put 5 Stone in dropper slot 0
		_ = dropper.inventory.SetItem(0, item.NewStack(Stone{}, 5))
		tx.SetBlock(dropperPos, dropper, nil)

		// Place target container (another Dropper) at {1, 64, 0}
		targetPos := cube.Pos{1, 64, 0}
		targetDropper := NewDropper()
		tx.SetBlock(targetPos, targetDropper, nil)

		// Execute drop: should transfer 1 Stone into target container
		dropper.Drop(dropperPos, tx)

		dItem, _ := dropper.inventory.Item(0)
		if dItem.Count() != 4 {
			t.Fatalf("expected dropper to have 4 items left, got %d", dItem.Count())
		}

		targetItem, _ := targetDropper.inventory.Item(0)
		if targetItem.Count() != 1 {
			t.Fatalf("expected target container to have received 1 item, got %d", targetItem.Count())
		}
	}).Wait(context.Background())
}

func TestDispenser_BucketDispense(t *testing.T) {
	w := newTestWorld()
	dispenserPos := cube.Pos{0, 64, 0}
	disp := NewDispenser()
	disp.Facing = cube.FaceEast
	targetPos := cube.Pos{1, 64, 0}

	w.Do(func(tx *world.Tx) {
		// Put water bucket in slot 0
		waterBucket := item.NewStack(item.Bucket{Content: item.LiquidBucketContent(Water{})}, 1)
		_ = disp.inventory.SetItem(0, waterBucket)
		tx.SetBlock(dispenserPos, disp, nil)

		// Dispense liquid into air at targetPos
		disp.Dispense(dispenserPos, tx)

		// Target should now contain water liquid
		if _, ok := tx.Liquid(targetPos); !ok {
			t.Fatal("expected water liquid placed at target position")
		}

		// Slot 0 in dispenser should now be empty bucket
		slotItem, _ := disp.inventory.Item(0)
		bucket, ok := slotItem.Item().(item.Bucket)
		if !ok || bucket.Content.String() != "" {
			t.Fatalf("expected empty bucket in slot 0, got %v", slotItem.Item())
		}

		// Dispense again: empty bucket scoops up the water!
		disp.Dispense(dispenserPos, tx)

		// Liquid at target should now be removed
		if _, ok := tx.Liquid(targetPos); ok {
			t.Fatal("expected water liquid scooped up from target position")
		}

		// Slot 0 should be water bucket again
		slotItem2, _ := disp.inventory.Item(0)
		bucket2, ok := slotItem2.Item().(item.Bucket)
		if !ok || bucket2.Content.String() != "water" {
			t.Fatalf("expected water bucket in slot 0 after scoop, got %v", slotItem2.Item())
		}
	}).Wait(context.Background())
}

func TestCrafter_ComparatorSignal(t *testing.T) {
	c := NewCrafter()

	// Initial empty crafter: signal 0
	if sig := c.CalculateComparatorSignal(); sig != 0 {
		t.Fatalf("expected empty signal 0, got %d", sig)
	}

	// Disable 3 slots: slots 0, 1, 2
	c.SetSlotDisabled(0, true)
	c.SetSlotDisabled(1, true)
	c.SetSlotDisabled(2, true)
	if sig := c.CalculateComparatorSignal(); sig != 3 {
		t.Fatalf("expected signal 3 for 3 disabled slots, got %d", sig)
	}

	// Add items to 2 enabled slots: slots 3 and 4
	_ = c.inventory.SetItem(3, item.NewStack(Stone{}, 1))
	_ = c.inventory.SetItem(4, item.NewStack(Stone{}, 1))
	if sig := c.CalculateComparatorSignal(); sig != 5 {
		t.Fatalf("expected signal 5 (3 disabled + 2 filled), got %d", sig)
	}

	// Fill remaining 4 slots
	for i := 5; i < 9; i++ {
		_ = c.inventory.SetItem(i, item.NewStack(Stone{}, 1))
	}
	if sig := c.CalculateComparatorSignal(); sig != 9 {
		t.Fatalf("expected full signal 9, got %d", sig)
	}
}

func TestCrafter_CraftingSynthesis(t *testing.T) {
	// Register a test recipe: 1 Stone -> 2 Sticks (shaped 1x1)
	recipe.Register(recipe.NewShaped(
		[]recipe.Item{item.NewStack(Stone{}, 1)},
		item.NewStack(item.Stick{}, 2),
		recipe.NewShape(1, 1),
		"crafting_table",
	))

	w := newTestWorld()
	crafterPos := cube.Pos{0, 64, 0}
	crafter := NewCrafter()
	crafter.Orientation = CrafterEastUp // Faces East

	w.Do(func(tx *world.Tx) {
		// Put 2 Stone in crafter slot 0
		_ = crafter.inventory.SetItem(0, item.NewStack(Stone{}, 2))
		tx.SetBlock(crafterPos, crafter, nil)

		// Place target container (Dropper) at East
		targetPos := cube.Pos{1, 64, 0}
		targetContainer := NewDropper()
		tx.SetBlock(targetPos, targetContainer, nil)

		// Execute crafting
		if !crafter.Craft(crafterPos, tx) {
			t.Fatal("expected crafting synthesis to succeed")
		}

		// Input in slot 0 should decrease from 2 to 1
		st, _ := crafter.inventory.Item(0)
		if st.Count() != 1 {
			t.Fatalf("expected remaining input count 1, got %d", st.Count())
		}

		// Target container should have received 2 Sticks
		out, _ := targetContainer.inventory.Item(0)
		if _, ok := out.Item().(item.Stick); !ok {
			t.Fatalf("expected target container to receive Stick, got %T", out.Item())
		}
	}).Wait(context.Background())
}

func BenchmarkObserver_Evaluation(b *testing.B) {
	obs := Observer{Facing: cube.FaceNorth, Powered: true}
	pos := cube.Pos{0, 64, 0}
	rearFace := cube.FaceSouth

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = obs.RedstonePower(pos, nil, rearFace)
	}
}

func BenchmarkCrafter_ComparatorSignal(b *testing.B) {
	c := NewCrafter()
	c.SetSlotDisabled(0, true)
	c.SetSlotDisabled(1, true)
	_ = c.inventory.SetItem(2, item.NewStack(Stone{}, 1))
	_ = c.inventory.SetItem(3, item.NewStack(Stone{}, 1))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = c.CalculateComparatorSignal()
	}
}

func TestAutomation_NilAndBoundarySafety(t *testing.T) {
	obs := Observer{Facing: cube.FaceNorth}
	disp := NewDispenser()
	drop := NewDropper()
	craft := NewCrafter()

	// 1. Nil Tx safety
	obs.NeighbourUpdateTick(cube.Pos{0, 0, 0}, cube.Pos{0, 0, 0}, nil)
	obs.ScheduledTick(cube.Pos{0, 0, 0}, nil, nil)

	disp.NeighbourUpdateTick(cube.Pos{0, 0, 0}, cube.Pos{0, 0, 0}, nil)
	disp.RedstonePowerAction(cube.Pos{0, 0, 0}, nil, 0, 15)
	disp.ScheduledTick(cube.Pos{0, 0, 0}, nil, nil)
	disp.Dispense(cube.Pos{0, 0, 0}, nil)
	disp.AddViewer(nil, nil, cube.Pos{0, 0, 0})
	disp.RemoveViewer(nil, nil, cube.Pos{0, 0, 0})

	drop.NeighbourUpdateTick(cube.Pos{0, 0, 0}, cube.Pos{0, 0, 0}, nil)
	drop.RedstonePowerAction(cube.Pos{0, 0, 0}, nil, 0, 15)
	drop.ScheduledTick(cube.Pos{0, 0, 0}, nil, nil)
	drop.Drop(cube.Pos{0, 0, 0}, nil)
	drop.AddViewer(nil, nil, cube.Pos{0, 0, 0})
	drop.RemoveViewer(nil, nil, cube.Pos{0, 0, 0})

	craft.NeighbourUpdateTick(cube.Pos{0, 0, 0}, cube.Pos{0, 0, 0}, nil)
	craft.RedstonePowerAction(cube.Pos{0, 0, 0}, nil, 0, 15)
	craft.ScheduledTick(cube.Pos{0, 0, 0}, nil, nil)
	craft.Craft(cube.Pos{0, 0, 0}, nil)
	craft.AddViewer(nil, nil, cube.Pos{0, 0, 0})
	craft.RemoveViewer(nil, nil, cube.Pos{0, 0, 0})

	// Slot bounds
	craft.SetSlotDisabled(-1, true)
	craft.SetSlotDisabled(10, true)
	if craft.IsSlotDisabled(-1) || craft.IsSlotDisabled(10) {
		t.Fatal("expected out of bounds slots to be false")
	}

	// 2. Out of bounds safety in real world
	w := newTestWorld()
	w.Do(func(tx *world.Tx) {
		r := tx.Range()
		outPos := cube.Pos{0, r.Max() + 10, 0}

		obs.NeighbourUpdateTick(outPos, outPos, tx)
		obs.ScheduledTick(outPos, tx, nil)

		disp.NeighbourUpdateTick(outPos, outPos, tx)
		disp.RedstonePowerAction(outPos, tx, 0, 15)
		disp.ScheduledTick(outPos, tx, nil)
		disp.Dispense(outPos, tx)

		drop.NeighbourUpdateTick(outPos, outPos, tx)
		drop.RedstonePowerAction(outPos, tx, 0, 15)
		drop.ScheduledTick(outPos, tx, nil)
		drop.Drop(outPos, tx)

		craft.NeighbourUpdateTick(outPos, outPos, tx)
		craft.RedstonePowerAction(outPos, tx, 0, 15)
		craft.ScheduledTick(outPos, tx, nil)
		craft.Craft(outPos, tx)
	}).Wait(context.Background())
}

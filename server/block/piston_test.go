package block

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func newTestWorld() *world.World {
	w := world.Config{Dim: world.Overworld, Synchronous: true}.New()
	_ = world.NewLoader(1, w, world.NopViewer{})
	return w
}

func TestPiston_EncodeDecode(t *testing.T) {
	for _, p := range allPistons() {
		name, properties := p.EncodeBlock()
		expectedName := "minecraft:piston"
		if p.(Piston).Sticky {
			expectedName = "minecraft:sticky_piston"
		}
		if name != expectedName {
			t.Fatalf("expected name %s, got %s", expectedName, name)
		}
		facing, ok := properties["facing_direction"].(int32)
		if !ok || facing < 0 || facing > 5 {
			t.Fatalf("invalid facing_direction property: %v", properties["facing_direction"])
		}
	}

	for _, a := range allPistonArms() {
		name, properties := a.EncodeBlock()
		expectedName := "minecraft:piston_arm_collision"
		if a.(PistonArm).Sticky {
			expectedName = "minecraft:sticky_piston_arm_collision"
		}
		if name != expectedName {
			t.Fatalf("expected name %s, got %s", expectedName, name)
		}
		facing, ok := properties["facing_direction"].(int32)
		if !ok || facing < 0 || facing > 5 {
			t.Fatalf("invalid facing_direction property: %v", properties["facing_direction"])
		}
	}

	honeyName, _ := Honey{}.EncodeBlock()
	if honeyName != "minecraft:honey_block" {
		t.Fatalf("expected minecraft:honey_block, got %s", honeyName)
	}
}

func TestPiston_PushSingleBlock(t *testing.T) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}
	stonePos := cube.Pos{1, 64, 0}
	piston := Piston{Facing: cube.FaceEast, Sticky: false}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, piston, nil)
		tx.SetBlock(stonePos, Stone{}, nil)

		// Extend piston pushing east
		if !piston.Extend(pistonPos, tx) {
			t.Fatal("failed to extend piston")
		}

		// Piston arm should be at stonePos
		if arm, ok := tx.Block(stonePos).(PistonArm); !ok || arm.Facing != cube.FaceEast {
			t.Fatalf("expected piston arm at %v, got %T", stonePos, tx.Block(stonePos))
		}

		// Stone should have moved to {2, 64, 0}
		movedStonePos := cube.Pos{2, 64, 0}
		if _, ok := tx.Block(movedStonePos).(Stone); !ok {
			t.Fatalf("expected stone at %v, got %T", movedStonePos, tx.Block(movedStonePos))
		}

		// Retract normal piston
		if !piston.Retract(pistonPos, tx) {
			t.Fatal("failed to retract piston")
		}

		// Arm is removed, stone stays at {2, 64, 0}
		if _, ok := tx.Block(stonePos).(Air); !ok {
			t.Fatalf("expected air at %v, got %T", stonePos, tx.Block(stonePos))
		}
		if _, ok := tx.Block(movedStonePos).(Stone); !ok {
			t.Fatalf("expected stone to stay at %v, got %T", movedStonePos, tx.Block(movedStonePos))
		}
	}).Wait(context.Background())
}

func TestPiston_Push12BlockLimit(t *testing.T) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}
	piston := Piston{Facing: cube.FaceEast}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, piston, nil)

		// Place exactly 12 stone blocks: {1, 64, 0} to {12, 64, 0}
		for i := 1; i <= 12; i++ {
			tx.SetBlock(cube.Pos{i, 64, 0}, Stone{}, nil)
		}

		if !piston.Extend(pistonPos, tx) {
			t.Fatal("expected 12 blocks to be pushed successfully")
		}
		if _, ok := tx.Block(cube.Pos{13, 64, 0}).(Stone); !ok {
			t.Fatalf("expected stone at 13th pos, got %T", tx.Block(cube.Pos{13, 64, 0}))
		}
	}).Wait(context.Background())

	// Test 13 blocks fails
	w2 := newTestWorld()
	w2.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, piston, nil)
		for i := 1; i <= 13; i++ {
			tx.SetBlock(cube.Pos{i, 64, 0}, Stone{}, nil)
		}

		if piston.Extend(pistonPos, tx) {
			t.Fatal("expected 13 blocks push to fail")
		}
		if piston.Extended(pistonPos, tx) {
			t.Fatal("piston should not be extended")
		}
	}).Wait(context.Background())
}

func TestPiston_ImmovableBlocks(t *testing.T) {
	immovables := []world.Block{
		Obsidian{},
		Obsidian{Crying: true},
		Bedrock{},
		ReinforcedDeepslate{},
		Barrier{},
	}

	for _, imm := range immovables {
		w := newTestWorld()
		pistonPos := cube.Pos{0, 64, 0}
		piston := Piston{Facing: cube.FaceEast}

		w.Do(func(tx *world.Tx) {
			tx.SetBlock(pistonPos, piston, nil)
			tx.SetBlock(cube.Pos{1, 64, 0}, Stone{}, nil)
			tx.SetBlock(cube.Pos{2, 64, 0}, imm, nil)

			if piston.Extend(pistonPos, tx) {
				t.Fatalf("expected push to be blocked by immovable block %T", imm)
			}
		}).Wait(context.Background())
	}
}

func TestPiston_DestroyableOnPush(t *testing.T) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}
	piston := Piston{Facing: cube.FaceEast}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, piston, nil)
		tx.SetBlock(cube.Pos{1, 64, 0}, Stone{}, nil)
		tx.SetBlock(cube.Pos{2, 64, 0}, Torch{}, nil)

		if !piston.Extend(pistonPos, tx) {
			t.Fatal("expected push to succeed through breakable torch")
		}

		// Stone should move to pos 2, replacing the torch
		if _, ok := tx.Block(cube.Pos{2, 64, 0}).(Stone); !ok {
			t.Fatalf("expected stone at pos 2, got %T", tx.Block(cube.Pos{2, 64, 0}))
		}
	}).Wait(context.Background())
}

func TestStickyPiston_PullSingleBlock(t *testing.T) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}
	piston := Piston{Facing: cube.FaceEast, Sticky: true}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, piston, nil)
		tx.SetBlock(cube.Pos{1, 64, 0}, Stone{}, nil)

		// Push stone to pos 2
		if !piston.Extend(pistonPos, tx) {
			t.Fatal("failed to extend sticky piston")
		}
		if _, ok := tx.Block(cube.Pos{2, 64, 0}).(Stone); !ok {
			t.Fatalf("expected stone at pos 2, got %T", tx.Block(cube.Pos{2, 64, 0}))
		}

		// Retract sticky piston: pulls stone back to pos 1
		if !piston.Retract(pistonPos, tx) {
			t.Fatal("failed to retract sticky piston")
		}
		if _, ok := tx.Block(cube.Pos{1, 64, 0}).(Stone); !ok {
			t.Fatalf("expected stone pulled back to pos 1, got %T", tx.Block(cube.Pos{1, 64, 0}))
		}
		if _, ok := tx.Block(cube.Pos{2, 64, 0}).(Air); !ok {
			t.Fatalf("expected pos 2 to be air, got %T", tx.Block(cube.Pos{2, 64, 0}))
		}
	}).Wait(context.Background())
}

func TestPiston_SlimeAdhesionNetwork(t *testing.T) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}
	piston := Piston{Facing: cube.FaceEast, Sticky: true}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, piston, nil)
		// Slime at pos 1, stone attached above slime at {1, 65, 0}
		tx.SetBlock(cube.Pos{1, 64, 0}, Slime{}, nil)
		tx.SetBlock(cube.Pos{1, 65, 0}, Stone{}, nil)

		if !piston.Extend(pistonPos, tx) {
			t.Fatal("failed to extend slime cluster")
		}

		// Slime moved to {2, 64, 0}, stone moved to {2, 65, 0}
		if _, ok := tx.Block(cube.Pos{2, 64, 0}).(Slime); !ok {
			t.Fatalf("expected slime at {2, 64, 0}, got %T", tx.Block(cube.Pos{2, 64, 0}))
		}
		if _, ok := tx.Block(cube.Pos{2, 65, 0}).(Stone); !ok {
			t.Fatalf("expected adhered stone at {2, 65, 0}, got %T", tx.Block(cube.Pos{2, 65, 0}))
		}

		// Retract sticky piston
		if !piston.Retract(pistonPos, tx) {
			t.Fatal("failed to retract slime cluster")
		}

		// Both slime and attached stone pulled back to pos 1
		if _, ok := tx.Block(cube.Pos{1, 64, 0}).(Slime); !ok {
			t.Fatalf("expected slime pulled back to {1, 64, 0}, got %T", tx.Block(cube.Pos{1, 64, 0}))
		}
		if _, ok := tx.Block(cube.Pos{1, 65, 0}).(Stone); !ok {
			t.Fatalf("expected stone pulled back to {1, 65, 0}, got %T", tx.Block(cube.Pos{1, 65, 0}))
		}
	}).Wait(context.Background())
}

func TestPiston_HoneyAdhesionNetwork(t *testing.T) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}
	piston := Piston{Facing: cube.FaceEast, Sticky: true}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, piston, nil)
		// Honey at pos 1, stone attached above honey at {1, 65, 0}
		tx.SetBlock(cube.Pos{1, 64, 0}, Honey{}, nil)
		tx.SetBlock(cube.Pos{1, 65, 0}, Stone{}, nil)

		if !piston.Extend(pistonPos, tx) {
			t.Fatal("failed to extend honey cluster")
		}

		// Honey moved to {2, 64, 0}, stone moved to {2, 65, 0}
		if _, ok := tx.Block(cube.Pos{2, 64, 0}).(Honey); !ok {
			t.Fatalf("expected honey at {2, 64, 0}, got %T", tx.Block(cube.Pos{2, 64, 0}))
		}
		if _, ok := tx.Block(cube.Pos{2, 65, 0}).(Stone); !ok {
			t.Fatalf("expected adhered stone at {2, 65, 0}, got %T", tx.Block(cube.Pos{2, 65, 0}))
		}

		// Retract sticky piston
		if !piston.Retract(pistonPos, tx) {
			t.Fatal("failed to retract honey cluster")
		}

		// Both honey and attached stone pulled back
		if _, ok := tx.Block(cube.Pos{1, 64, 0}).(Honey); !ok {
			t.Fatalf("expected honey pulled back to {1, 64, 0}, got %T", tx.Block(cube.Pos{1, 64, 0}))
		}
		if _, ok := tx.Block(cube.Pos{1, 65, 0}).(Stone); !ok {
			t.Fatalf("expected stone pulled back to {1, 65, 0}, got %T", tx.Block(cube.Pos{1, 65, 0}))
		}
	}).Wait(context.Background())
}

func TestPiston_SlimeAndHoneyNonAdhesion(t *testing.T) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}
	piston := Piston{Facing: cube.FaceEast, Sticky: true}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, piston, nil)
		// Slime in front of piston at pos 1
		tx.SetBlock(cube.Pos{1, 64, 0}, Slime{}, nil)
		// Honey adjacent to slime at {1, 65, 0}
		tx.SetBlock(cube.Pos{1, 65, 0}, Honey{}, nil)

		if !piston.Extend(pistonPos, tx) {
			t.Fatal("failed to extend piston")
		}

		// Slime should move to {2, 64, 0}
		if _, ok := tx.Block(cube.Pos{2, 64, 0}).(Slime); !ok {
			t.Fatalf("expected slime at {2, 64, 0}, got %T", tx.Block(cube.Pos{2, 64, 0}))
		}

		// Honey should NOT have moved! Honey does not adhere to slime!
		if _, ok := tx.Block(cube.Pos{1, 65, 0}).(Honey); !ok {
			t.Fatalf("expected honey to stay at {1, 65, 0}, got %T", tx.Block(cube.Pos{1, 65, 0}))
		}
	}).Wait(context.Background())
}

func TestPiston_GlazedTerracottaNonAdhesion(t *testing.T) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}
	piston := Piston{Facing: cube.FaceEast, Sticky: true}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, piston, nil)
		tx.SetBlock(cube.Pos{1, 64, 0}, Slime{}, nil)
		tx.SetBlock(cube.Pos{1, 65, 0}, GlazedTerracotta{}, nil)

		if !piston.Extend(pistonPos, tx) {
			t.Fatal("failed to extend piston")
		}

		// Glazed terracotta does NOT adhere to slime!
		if _, ok := tx.Block(cube.Pos{1, 65, 0}).(GlazedTerracotta); !ok {
			t.Fatalf("expected glazed terracotta to stay at {1, 65, 0}, got %T", tx.Block(cube.Pos{1, 65, 0}))
		}
	}).Wait(context.Background())
}

func TestPiston_FlyingMachineSimulation(t *testing.T) {
	// A classic 2-engine flying machine:
	// PistonA at {0, 64, 0} facing East (sticky) pushing Slime at {1, 64, 0}
	// Attached to Slime is PistonB at {1, 64, 1} facing West (sticky) pulling Honey at {0, 64, 1}
	// When PistonA extends, it moves Slime + PistonB forward by 1 block.
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}
	pistonA := Piston{Facing: cube.FaceEast, Sticky: true}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pistonPos, pistonA, nil)
		tx.SetBlock(cube.Pos{1, 64, 0}, Slime{}, nil)
		tx.SetBlock(cube.Pos{1, 64, 1}, Piston{Facing: cube.FaceWest, Sticky: true}, nil)

		if !pistonA.Extend(pistonPos, tx) {
			t.Fatal("flying machine step 1 extend failed")
		}

		// Slime and attached PistonB both moved East by 1
		if _, ok := tx.Block(cube.Pos{2, 64, 0}).(Slime); !ok {
			t.Fatalf("expected slime at {2, 64, 0}, got %T", tx.Block(cube.Pos{2, 64, 0}))
		}
		if pB, ok := tx.Block(cube.Pos{2, 64, 1}).(Piston); !ok || pB.Facing != cube.FaceWest {
			t.Fatalf("expected pistonB at {2, 64, 1}, got %v", tx.Block(cube.Pos{2, 64, 1}))
		}
	}).Wait(context.Background())
}

func BenchmarkPiston_PushChain(b *testing.B) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}

	w.Do(func(tx *world.Tx) {
		for i := 1; i <= 10; i++ {
			tx.SetBlock(cube.Pos{i, 64, 0}, Stone{}, nil)
		}
		solver := NewPistonSolver(tx, pistonPos, cube.FaceEast, false)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			solver.visited = make(map[cube.Pos]bool)
			solver.toMove = solver.toMove[:0]
			solver.toDestroy = solver.toDestroy[:0]
			_ = solver.calculatePush()
		}
	}).Wait(context.Background())
}

func BenchmarkPiston_AdhesionNetwork(b *testing.B) {
	w := newTestWorld()
	pistonPos := cube.Pos{0, 64, 0}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{1, 64, 0}, Slime{}, nil)
		tx.SetBlock(cube.Pos{1, 65, 0}, Stone{}, nil)
		tx.SetBlock(cube.Pos{1, 63, 0}, Stone{}, nil)
		tx.SetBlock(cube.Pos{1, 64, 1}, Stone{}, nil)
		solver := NewPistonSolver(tx, pistonPos, cube.FaceEast, false)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			solver.visited = make(map[cube.Pos]bool)
			solver.toMove = solver.toMove[:0]
			solver.toDestroy = solver.toDestroy[:0]
			_ = solver.calculatePush()
		}
	}).Wait(context.Background())
}

func TestPiston_NilAndBoundarySafety(t *testing.T) {
	piston := Piston{Facing: cube.FaceEast}
	arm := PistonArm{Facing: cube.FaceEast}

	// 1. Nil transaction safety
	if piston.Extended(cube.Pos{0, 0, 0}, nil) {
		t.Fatal("expected Extended with nil tx to return false")
	}
	if piston.Extend(cube.Pos{0, 0, 0}, nil) {
		t.Fatal("expected Extend with nil tx to return false")
	}
	if piston.Retract(cube.Pos{0, 0, 0}, nil) {
		t.Fatal("expected Retract with nil tx to return false")
	}
	piston.NeighbourUpdateTick(cube.Pos{0, 0, 0}, cube.Pos{0, 0, 0}, nil)
	piston.ScheduledTick(cube.Pos{0, 0, 0}, nil, nil)
	arm.NeighbourUpdateTick(cube.Pos{0, 0, 0}, cube.Pos{0, 0, 0}, nil)

	solver := NewPistonSolver(nil, cube.Pos{0, 0, 0}, cube.FaceEast, false)
	if solver.Solve() {
		t.Fatal("expected Solve with nil tx to return false")
	}

	// 2. Honey nil distance safety
	Honey{}.EntityLand(cube.Pos{0, 64, 0}, nil, nil, nil)

	// 3. Out-of-bounds safety in real world
	w := newTestWorld()
	w.Do(func(tx *world.Tx) {
		r := tx.Range()
		outPos := cube.Pos{0, r.Max() + 10, 0}
		if piston.Extend(outPos, tx) {
			t.Fatal("expected Extend out of bounds to return false")
		}
		if piston.Retract(outPos, tx) {
			t.Fatal("expected Retract out of bounds to return false")
		}
		if piston.Extended(outPos, tx) {
			t.Fatal("expected Extended out of bounds to return false")
		}

		// Container immobility check
		chestPos := cube.Pos{1, 64, 0}
		tx.SetBlock(cube.Pos{0, 64, 0}, Piston{Facing: cube.FaceEast}, nil)
		tx.SetBlock(chestPos, NewChest(), nil)
		p := Piston{Facing: cube.FaceEast}
		if p.Extend(cube.Pos{0, 64, 0}, tx) {
			t.Fatal("expected piston extend into container chest to be blocked")
		}
	}).Wait(context.Background())
}

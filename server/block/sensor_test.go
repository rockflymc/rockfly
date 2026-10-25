package block

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func TestSensor_EncodeDecode(t *testing.T) {
	// 1. Daylight detector: 32 permutations (2 inverted x 16 signals)
	for _, d := range allDaylightDetectors() {
		name, properties := d.EncodeBlock()
		expectedName := "minecraft:daylight_detector"
		if d.(DaylightDetector).Inverted {
			expectedName = "minecraft:daylight_detector_inverted"
		}
		if name != expectedName {
			t.Fatalf("expected name %s, got %s", expectedName, name)
		}
		sig, ok := properties["redstone_signal"].(int32)
		if !ok || sig < 0 || sig > 15 {
			t.Fatalf("invalid redstone_signal property: %v", properties["redstone_signal"])
		}
	}

	// 2. Target: 1 state (empty map)
	targetName, targetProps := Target{}.EncodeBlock()
	if targetName != "minecraft:target" {
		t.Fatalf("expected minecraft:target, got %s", targetName)
	}
	if len(targetProps) != 0 {
		t.Fatalf("expected empty properties for target, got %v", targetProps)
	}

	// 3. Trapped Chest: 4 directions
	for _, tc := range allTrappedChests() {
		name, properties := tc.EncodeBlock()
		if name != "minecraft:trapped_chest" {
			t.Fatalf("expected minecraft:trapped_chest, got %s", name)
		}
		dir, ok := properties["minecraft:cardinal_direction"].(string)
		if !ok || dir == "" {
			t.Fatalf("invalid cardinal_direction: %v", properties["minecraft:cardinal_direction"])
		}
	}

	// 4. Sculk Sensor: 3 phases standard + 12 calibrated = 15 total
	for _, s := range allSculkSensors() {
		name, properties := s.EncodeBlock()
		expectedName := "minecraft:sculk_sensor"
		if s.(SculkSensor).Calibrated {
			expectedName = "minecraft:calibrated_sculk_sensor"
		}
		if name != expectedName {
			t.Fatalf("expected %s, got %s", expectedName, name)
		}
		phase, ok := properties["sculk_sensor_phase"].(int32)
		if !ok || phase < 0 || phase > 2 {
			t.Fatalf("invalid sculk_sensor_phase: %v", properties["sculk_sensor_phase"])
		}
	}

	// 5. Tripwire Hook: 16 permutations (4 directions x 2 attached x 2 powered)
	for _, h := range allTripwireHooks() {
		name, properties := h.EncodeBlock()
		if name != "minecraft:tripwire_hook" {
			t.Fatalf("expected minecraft:tripwire_hook, got %s", name)
		}
		dir, ok := properties["direction"].(int32)
		if !ok || dir < 0 || dir > 3 {
			t.Fatalf("invalid direction: %v", properties["direction"])
		}
	}
}

func TestDaylightDetector_SolarCalculation(t *testing.T) {
	// Midday: tick 6000
	middayNormal := CalculateDaylightPower(6000, false)
	if middayNormal != 15 {
		t.Fatalf("expected midday normal signal 15, got %d", middayNormal)
	}
	middayInverted := CalculateDaylightPower(6000, true)
	if middayInverted != 0 {
		t.Fatalf("expected midday inverted signal 0, got %d", middayInverted)
	}

	// Midnight: tick 18000
	midnightNormal := CalculateDaylightPower(18000, false)
	if midnightNormal != 0 {
		t.Fatalf("expected midnight normal signal 0, got %d", midnightNormal)
	}
	midnightInverted := CalculateDaylightPower(18000, true)
	if midnightInverted != 15 {
		t.Fatalf("expected midnight inverted signal 15, got %d", midnightInverted)
	}

	// Sunset (tick 12000)
	sunsetNormal := CalculateDaylightPower(12000, false)
	if sunsetNormal != 0 {
		t.Fatalf("expected sunset normal signal 0, got %d", sunsetNormal)
	}
	sunsetInverted := CalculateDaylightPower(12000, true)
	if sunsetInverted != 15 {
		t.Fatalf("expected sunset inverted signal 15, got %d", sunsetInverted)
	}

	// Interaction test
	w := newTestWorld()
	pos := cube.Pos{0, 64, 0}
	d := DaylightDetector{Inverted: false, Power: 15}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, d, nil)
		d.Activate(pos, cube.FaceUp, tx, nil, nil)

		current, _ := tx.Block(pos).(DaylightDetector)
		if !current.Inverted {
			t.Fatal("expected daylight detector to be inverted after activation")
		}
	}).Wait(context.Background())
}

func TestTarget_BullseyeAndRings(t *testing.T) {
	pos := cube.Pos{0, 64, 0}
	center := pos.Vec3Centre()

	// Exact bullseye hit
	pBullseye := CalculateTargetPower(center, pos, cube.FaceNorth)
	if pBullseye != 15 {
		t.Fatalf("expected bullseye power 15, got %d", pBullseye)
	}

	// Edge hit at distance 0.35 from center
	edgePos := center.Add(mgl64.Vec3{0.35, 0, 0})
	pEdge := CalculateTargetPower(edgePos, pos, cube.FaceNorth)
	if pEdge >= 15 || pEdge < 1 {
		t.Fatalf("expected edge power between 1 and 14, got %d", pEdge)
	}

	// Full hit and scheduled tick cycle
	w := newTestWorld()
	target := Target{}
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, target, nil)
		power := target.Hit(pos, tx, center, cube.FaceNorth, true)
		if power != 15 {
			t.Fatalf("expected hit power 15, got %d", power)
		}

		hitTarget, _ := tx.Block(pos).(Target)
		if p := hitTarget.RedstonePower(pos, tx, cube.FaceSouth); p != 15 {
			t.Fatalf("expected redstone emission 15, got %d", p)
		}

		// Scheduled tick expires duration
		hitTarget.ScheduledTick(pos, tx, nil)
		offTarget, _ := tx.Block(pos).(Target)
		if p := offTarget.RedstonePower(pos, tx, cube.FaceSouth); p != 0 {
			t.Fatalf("expected power reset to 0, got %d", p)
		}
	}).Wait(context.Background())
}

type dummyViewer struct {
	world.NopViewer
	id int
}

func (dummyViewer) ViewSlotChange(int, item.Stack) {}

func TestTrappedChest_ViewerRedstonePower(t *testing.T) {
	w := newTestWorld()
	pos := cube.Pos{0, 64, 0}
	chest := NewTrappedChest()

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, chest, nil)

		// 0 viewers: power 0
		if p := chest.RedstonePower(pos, tx, cube.FaceNorth); p != 0 {
			t.Fatalf("expected 0 power initially, got %d", p)
		}

		// 1 viewer added
		v1 := dummyViewer{id: 1}
		chest.AddViewer(v1, tx, pos)
		if p := chest.RedstonePower(pos, tx, cube.FaceNorth); p != 1 {
			t.Fatalf("expected power 1 with 1 viewer, got %d", p)
		}
		// Strong power downward
		if p := chest.RedstoneStrongPower(pos, tx, cube.FaceDown); p != 1 {
			t.Fatalf("expected downward strong power 1, got %d", p)
		}

		// 2nd viewer added
		v2 := dummyViewer{id: 2}
		chest.AddViewer(v2, tx, pos)
		if p := chest.RedstonePower(pos, tx, cube.FaceNorth); p != 2 {
			t.Fatalf("expected power 2 with 2 viewers, got %d", p)
		}

		// Remove viewer
		chest.RemoveViewer(v1, tx, pos)
		if p := chest.RedstonePower(pos, tx, cube.FaceNorth); p != 1 {
			t.Fatalf("expected power 1 after removing 1 viewer, got %d", p)
		}
		chest.RemoveViewer(v2, tx, pos)
		if p := chest.RedstonePower(pos, tx, cube.FaceNorth); p != 0 {
			t.Fatalf("expected power 0 after removing all viewers, got %d", p)
		}
	}).Wait(context.Background())
}

func TestSculkSensor_VibrationAndCalibration(t *testing.T) {
	w := newTestWorld()
	pos := cube.Pos{0, 64, 0}
	sensor := SculkSensor{Phase: 0}

	w.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, sensor, nil)

		// Trigger standard sensor with explosion (freq 15)
		if !sensor.Trigger(pos, tx, VibrationExplosion) {
			t.Fatal("expected vibration trigger to succeed")
		}

		active, _ := tx.Block(pos).(SculkSensor)
		if active.Phase != 1 {
			t.Fatalf("expected phase 1, got phase %d", active.Phase)
		}
		if p := active.RedstonePower(pos, tx, cube.FaceUp); p != 15 {
			t.Fatalf("expected redstone emission 15, got %d", p)
		}

		// Advance to cooldown
		active.ScheduledTick(pos, tx, nil)
		cooldown, _ := tx.Block(pos).(SculkSensor)
		if cooldown.Phase != 2 {
			t.Fatalf("expected phase 2 cooldown, got %d", cooldown.Phase)
		}

		// Advance to inactive
		cooldown.ScheduledTick(pos, tx, nil)
		inactive, _ := tx.Block(pos).(SculkSensor)
		if inactive.Phase != 0 {
			t.Fatalf("expected phase 0 inactive, got %d", inactive.Phase)
		}
	}).Wait(context.Background())
}

func TestTripwireHook_LineConnection(t *testing.T) {
	w := newTestWorld()
	posA := cube.Pos{0, 64, 0}
	posB := cube.Pos{4, 64, 0}

	hookA := TripwireHook{Facing: cube.East}
	hookB := TripwireHook{Facing: cube.West}

	w.Do(func(tx *world.Tx) {
		// Wall for hook A at {-1, 64, 0} and hook B at {5, 64, 0}
		tx.SetBlock(cube.Pos{-1, 64, 0}, Stone{}, nil)
		tx.SetBlock(cube.Pos{5, 64, 0}, Stone{}, nil)

		tx.SetBlock(posA, hookA, nil)
		tx.SetBlock(posB, hookB, nil)

		// Place string at 1, 2, 3
		for x := 1; x <= 3; x++ {
			tx.SetBlock(cube.Pos{x, 64, 0}, String{}, nil)
		}

		// Check line from Hook A
		attached, powered := hookA.CheckLine(posA, tx)
		if !attached {
			t.Fatal("expected tripwire line to be attached")
		}
		if powered {
			t.Fatal("expected tripwire line not to be powered initially")
		}

		hA, _ := tx.Block(posA).(TripwireHook)
		hB, _ := tx.Block(posB).(TripwireHook)
		if !hA.Attached || !hB.Attached {
			t.Fatal("expected both hooks to be attached")
		}

		for x := 1; x <= 3; x++ {
			str, _ := tx.Block(cube.Pos{x, 64, 0}).(String)
			if !str.Attached {
				t.Fatalf("expected string at %d to be attached", x)
			}
		}
	}).Wait(context.Background())
}

func BenchmarkDaylightDetector_SolarCalculation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CalculateDaylightPower(6000+i, false)
	}
}

func BenchmarkTarget_HitCalculation(b *testing.B) {
	pos := cube.Pos{0, 64, 0}
	impact := pos.Vec3Centre().Add(mgl64.Vec3{0.12, 0.05, 0})
	face := cube.FaceNorth

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CalculateTargetPower(impact, pos, face)
	}
}

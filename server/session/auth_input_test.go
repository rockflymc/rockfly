package session

import (
	"context"
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

type dummyControllable struct {
	Controllable
	pos      mgl64.Vec3
	id       uuid.UUID
	gameMode world.GameMode
	gliding  bool
	flying   bool
}

func (d *dummyControllable) Position() mgl64.Vec3              { return d.pos }
func (d *dummyControllable) UUID() uuid.UUID                   { return d.id }
func (d *dummyControllable) GameMode() world.GameMode          { return d.gameMode }
func (d *dummyControllable) Gliding() bool                     { return d.gliding }
func (d *dummyControllable) Flying() bool                      { return d.flying }
func (d *dummyControllable) Swimming() bool                    { return false }
func (d *dummyControllable) FlightSpeed() float64              { return 1.5 }
func (d *dummyControllable) Move(mgl64.Vec3, float64, float64) {}

func TestMovementValidator_NormalWalking(t *testing.T) {
	v := NewMovementValidator(DefaultMovementValidationConfig())
	c := &dummyControllable{id: uuid.New(), pos: mgl64.Vec3{0, 64, 0}, gameMode: world.GameModeSurvival}

	oldPos := mgl64.Vec3{0, 64, 0}
	newPos := mgl64.Vec3{0.3, 64, 0.2} // 0.36 blocks distance

	valid, reason := v.Validate(c, nil, oldPos, newPos)
	if !valid {
		t.Fatalf("expected normal walk to be valid, got reason: %s", reason)
	}
}

func TestMovementValidator_SpeedHackRejection(t *testing.T) {
	v := NewMovementValidator(DefaultMovementValidationConfig())
	c := &dummyControllable{id: uuid.New(), pos: mgl64.Vec3{0, 64, 0}, gameMode: world.GameModeSurvival}

	oldPos := mgl64.Vec3{0, 64, 0}
	newPos := mgl64.Vec3{50, 64, 0} // 50 blocks distance spoofed

	valid, reason := v.Validate(c, nil, oldPos, newPos)
	if valid {
		t.Fatal("expected 50-block speed hack to be rejected")
	}
	if reason == "" {
		t.Fatal("expected non-empty rejection reason")
	}
}

func TestMovementValidator_PhaseNoClipRejection(t *testing.T) {
	v := NewMovementValidator(DefaultMovementValidationConfig())
	c := &dummyControllable{id: uuid.New(), pos: mgl64.Vec3{0, 64, 0}, gameMode: world.GameModeSurvival}

	w := (world.Config{Dim: world.Overworld, Synchronous: true}).New()
	defer w.Close()

	targetPos := cube.Pos{1, 64, 0}
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(targetPos, block.Obsidian{}, nil)
		tx.SetBlock(targetPos.Side(cube.FaceUp), block.Obsidian{}, nil)

		oldPos := mgl64.Vec3{0, 64, 0}
		newPos := mgl64.Vec3{1.5, 64, 0.5} // Inside solid obsidian

		valid, reason := v.Validate(c, tx, oldPos, newPos)
		if valid {
			t.Fatal("expected movement through solid obsidian to be rejected as phase hack")
		}
		if reason == "" {
			t.Fatal("expected non-empty rejection reason for phase hack")
		}
	}).Wait(context.Background())
}

func TestMovementValidator_FlightHackRejection(t *testing.T) {
	v := NewMovementValidator(DefaultMovementValidationConfig())
	c := &dummyControllable{id: uuid.New(), pos: mgl64.Vec3{0, 100, 0}, gameMode: world.GameModeSurvival, flying: false}

	w := (world.Config{Dim: world.Overworld, Synchronous: true}).New()
	defer w.Close()

	w.Do(func(tx *world.Tx) {
		// Simulate player hovering/ascending in mid-air for 90 ticks
		for i := 0; i < 90; i++ {
			oldPos := mgl64.Vec3{0, 100.0 + float64(i)*0.1, 0}
			newPos := mgl64.Vec3{0, 100.0 + float64(i+1)*0.1, 0}
			valid, _ := v.Validate(c, tx, oldPos, newPos)
			if i > 80 && valid {
				t.Fatalf("expected fly hack to be rejected after 80 air ticks, passed at tick %d", i)
			}
		}
	}).Wait(context.Background())
}

func TestMovementValidator_SpectatorBypass(t *testing.T) {
	v := NewMovementValidator(DefaultMovementValidationConfig())
	c := &dummyControllable{id: uuid.New(), pos: mgl64.Vec3{0, 64, 0}, gameMode: world.GameModeSpectator}

	oldPos := mgl64.Vec3{0, 64, 0}
	newPos := mgl64.Vec3{100, 200, 100} // Massive movement in spectator

	valid, _ := v.Validate(c, nil, oldPos, newPos)
	if !valid {
		t.Fatal("expected spectator mode to bypass movement validation")
	}
}

func BenchmarkMovementValidator_Validate(b *testing.B) {
	v := NewMovementValidator(DefaultMovementValidationConfig())
	c := &dummyControllable{id: uuid.New(), pos: mgl64.Vec3{0, 64, 0}, gameMode: world.GameModeSurvival}

	oldPos := mgl64.Vec3{0, 64, 0}
	newPos := mgl64.Vec3{0.3, 64, 0.2}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = v.Validate(c, nil, oldPos, newPos)
	}
}

func TestMovementValidator_NaNAndNilSafety(t *testing.T) {
	v := NewMovementValidator(DefaultMovementValidationConfig())
	c := &dummyControllable{id: uuid.New(), pos: mgl64.Vec3{0, 64, 0}, gameMode: world.GameModeSurvival}

	// 1. NaN coordinate injection
	valid, reason := v.Validate(c, nil, mgl64.Vec3{0, 64, 0}, mgl64.Vec3{0, 64, math.NaN()})
	if valid {
		t.Fatal("expected NaN coordinate injection to be rejected")
	}
	if reason == "" {
		t.Fatal("expected rejection reason for NaN injection")
	}

	// 2. Inf coordinate injection
	valid, reason = v.Validate(c, nil, mgl64.Vec3{0, 64, 0}, mgl64.Vec3{math.Inf(1), 64, 0})
	if valid {
		t.Fatal("expected Inf coordinate injection to be rejected")
	}

	// 3. Nil validator
	var nilValidator *MovementValidator
	valid, _ = nilValidator.Validate(c, nil, mgl64.Vec3{0, 0, 0}, mgl64.Vec3{1, 1, 1})
	if !valid {
		t.Fatal("expected true on nil validator")
	}
	nilValidator.Clear(uuid.New())
	nilValidator.Reconcile(nil, nil, mgl64.Vec3{0, 0, 0})

	// 4. Zero-valued validator
	var zeroValidator MovementValidator
	zeroValidator.Clear(uuid.New())
}

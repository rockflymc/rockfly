package ai

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

type mockMob struct {
	pos         mgl64.Vec3
	vel         mgl64.Vec3
	rot         cube.Rotation
	speed       float64
	health      float64
	dead        bool
	onFire      time.Duration
	target      Mob
	panicking   bool
	submerged   bool
	attackDealt float64
}

func newMockMob() *mockMob {
	return &mockMob{
		pos:    mgl64.Vec3{0, 0, 0},
		speed:  0.1,
		health: 20.0,
	}
}

func (m *mockMob) Position() mgl64.Vec3          { return m.pos }
func (m *mockMob) SetPosition(pos mgl64.Vec3)    { m.pos = pos }
func (m *mockMob) Velocity() mgl64.Vec3          { return m.vel }
func (m *mockMob) SetVelocity(vel mgl64.Vec3)    { m.vel = vel }
func (m *mockMob) Rotation() cube.Rotation       { return m.rot }
func (m *mockMob) SetRotation(rot cube.Rotation) { m.rot = rot }
func (m *mockMob) Speed() float64                { return m.speed }
func (m *mockMob) SetSpeed(s float64)            { m.speed = s }
func (m *mockMob) Health() float64               { return m.health }
func (m *mockMob) Dead() bool                    { return m.dead }
func (m *mockMob) OnFireDuration() time.Duration { return m.onFire }
func (m *mockMob) Target() Mob                   { return m.target }
func (m *mockMob) SetTarget(t Mob)               { m.target = t }

func TestGoalSelector_PanicPreemptsStroll(t *testing.T) {
	mob := newMockMob()
	selector := NewGoalSelector()

	stroll := NewRandomStrollGoal(mob, 1) // Always ready
	panicGoal := NewPanicGoal(mob, 1.5, func() bool {
		return mob.panicking
	})

	// Register stroll at Priority 5, panic at Priority 1
	selector.AddGoal(5, stroll)
	selector.AddGoal(1, panicGoal)

	// Tick 1: Mob is peaceful, stroll should activate
	selector.Tick()
	running := selector.RunningGoals()
	if len(running) != 1 || running[0] != stroll {
		t.Fatalf("expected RandomStrollGoal to run initially")
	}

	// Mob takes damage: trigger panic
	mob.panicking = true

	// Tick 2: PanicGoal (Priority 1) should preempt RandomStrollGoal (Priority 5)
	selector.Tick()
	running = selector.RunningGoals()
	if len(running) != 1 || running[0] != panicGoal {
		t.Fatalf("expected PanicGoal to preempt RandomStrollGoal, running: %v", running)
	}

	// Verify panic speed boost applied (0.1 * 1.5 = 0.15)
	if mob.Speed() < 0.149 || mob.Speed() > 0.151 {
		t.Fatalf("expected mob speed 0.15 during panic, got %v", mob.Speed())
	}
}

func TestGoalSelector_FloatGoal(t *testing.T) {
	mob := newMockMob()
	selector := NewGoalSelector()

	floatGoal := NewFloatGoal(mob, func() bool {
		return mob.submerged
	})
	selector.AddGoal(0, floatGoal)

	// Out of water: does not activate
	selector.Tick()
	if len(selector.RunningGoals()) != 0 {
		t.Fatalf("expected FloatGoal inactive out of water")
	}

	// Submerged in water: activates and propels upward
	mob.submerged = true
	selector.Tick()
	if len(selector.RunningGoals()) != 1 {
		t.Fatalf("expected FloatGoal to activate when submerged")
	}
	if mob.Velocity()[1] < 0.25 {
		t.Fatalf("expected upward velocity >= 0.25, got %v", mob.Velocity()[1])
	}
}

func TestGoalSelector_MeleeAttack(t *testing.T) {
	attacker := newMockMob()
	target := newMockMob()
	target.SetPosition(mgl64.Vec3{1.5, 0, 0}) // Within 2.0 block attack reach

	attacker.SetTarget(target)

	selector := NewGoalSelector()
	var damageDealt float64
	melee := NewMeleeAttackGoal(attacker, attacker, 2.0, 4.0, func(victim Mob, dmg float64) {
		damageDealt = dmg
	})
	selector.AddGoal(2, melee)

	// Tick: strikes target within reach
	selector.Tick()
	if damageDealt != 4.0 {
		t.Fatalf("expected 4.0 melee damage dealt, got %v", damageDealt)
	}

	// Move target away: (10, 0, 0)
	target.SetPosition(mgl64.Vec3{10, 0, 0})
	selector.Tick()

	// Velocity should point in +X direction towards target
	if attacker.Velocity()[0] <= 0 {
		t.Fatalf("expected attacker to move toward distant target, vel=%v", attacker.Velocity())
	}
}

func TestGoalSelector_TargetAcquisition(t *testing.T) {
	mob := newMockMob()
	enemy := newMockMob()
	enemy.SetPosition(mgl64.Vec3{5, 0, 0})

	targetSelector := NewGoalSelector()
	nearestTarget := NewNearestAttackableTargetGoal(mob, mob, 10.0, func(radius float64) Mob {
		return enemy
	})
	targetSelector.AddGoal(1, nearestTarget)

	targetSelector.Tick()
	if mob.Target() != enemy {
		t.Fatalf("expected enemy to be acquired as active target")
	}

	// Target dies
	enemy.dead = true
	targetSelector.Tick()
	if mob.Target() != nil {
		t.Fatalf("expected dead target to be dropped")
	}
}

func TestGoalSelector_HurtByTarget(t *testing.T) {
	mob := newMockMob()
	attacker := newMockMob()

	var lastAttacker Mob
	hurtByTarget := NewHurtByTargetGoal(mob, mob, func() Mob {
		return lastAttacker
	})

	selector := NewGoalSelector()
	selector.AddGoal(1, hurtByTarget)

	// Initially no attacker
	selector.Tick()
	if mob.Target() != nil {
		t.Fatalf("expected no target initially")
	}

	// Attacked by attacker
	lastAttacker = attacker
	selector.Tick()
	if mob.Target() != attacker {
		t.Fatalf("expected retaliation target to become attacker")
	}
}

func BenchmarkGoalSelector_Tick(b *testing.B) {
	mob := newMockMob()
	selector := NewGoalSelector()

	selector.AddGoal(0, NewFloatGoal(mob, func() bool { return false }))
	selector.AddGoal(1, NewPanicGoal(mob, 1.5, func() bool { return false }))
	selector.AddGoal(2, NewMeleeAttackGoal(mob, mob, 2.0, 3.0, nil))
	selector.AddGoal(5, NewRandomStrollGoal(mob, 10))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		selector.Tick()
	}
}

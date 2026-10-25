package combat

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Shield represents a handheld defensive tool that completely negates frontal melee and projectile
// attacks when blocking, but can be temporarily disabled for 5 seconds by axe strikes.
type Shield struct {
	disableTicks *atomic.Int32
}

// NewShield initializes an active Shield.
func NewShield() Shield {
	var a atomic.Int32
	return Shield{
		disableTicks: &a,
	}
}

// ensureInit guarantees that the internal atomic pointer is non-nil.
func (s *Shield) ensureInit() {
	if s.disableTicks == nil {
		var a atomic.Int32
		s.disableTicks = &a
	}
}

// Disabled returns true if the shield is currently recovering from an axe disable cooldown.
func (s Shield) Disabled() bool {
	if s.disableTicks == nil {
		return false
	}
	return s.disableTicks.Load() > 0
}

// DisableTicks returns the remaining disable cooldown in ticks.
func (s Shield) DisableTicks() int32 {
	if s.disableTicks == nil {
		return 0
	}
	return s.disableTicks.Load()
}

// Disable imposes a cooldown disabling the shield for the specified duration.
func (s Shield) Disable(duration time.Duration) {
	s.ensureInit()
	ticks := int32(duration.Milliseconds() / 50)
	s.disableTicks.Store(ticks)
}

// Tick decrements disable ticks each server tick.
func (s Shield) Tick() {
	if s.disableTicks != nil && s.disableTicks.Load() > 0 {
		s.disableTicks.Add(-1)
	}
}

// InBlockingArc verifies whether an incoming attack originates within the defender's 180-degree frontal blocking arc.
func InBlockingArc(attackerPos, defenderPos mgl64.Vec3, defenderYaw float64) bool {
	diff := attackerPos.Sub(defenderPos)
	diffXZ := mgl64.Vec3{diff[0], 0, diff[2]}
	if diffXZ.Len() < 0.001 {
		return true // Point-blank attack is blocked
	}

	// Calculate defender look vector from yaw
	yawRad := -defenderYaw * math.Pi / 180.0
	lookDir := mgl64.Vec3{-math.Sin(yawRad), 0, math.Cos(yawRad)}

	dot := lookDir.Dot(diffXZ.Normalize())
	// Dot product >= 0 corresponds to a 180-degree forward hemisphere
	return dot >= 0
}

// BlockDamage evaluates damage negation. If attacker is in frontal arc and shield is not disabled,
// blocks 100% of damage and checks for axe disable.
func (s Shield) BlockDamage(damage float64, src world.DamageSource, attackerPos, defenderPos mgl64.Vec3, defenderYaw float64, isAxe bool) (blocked float64, remaining float64, disabled bool) {
	if s.Disabled() {
		return 0, damage, false
	}

	if !InBlockingArc(attackerPos, defenderPos, defenderYaw) {
		return 0, damage, false
	}

	// 100% frontal damage blocked!
	if isAxe {
		s.Disable(5 * time.Second) // 5 seconds axe disable
		return damage, 0, true
	}

	return damage, 0, false
}

// MaxCount returns 1 as shields are non-stackable.
func (Shield) MaxCount() int {
	return 1
}

// DurabilityInfo returns shield durability configuration (336 uses).
func (Shield) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{
		MaxDurability: 336,
		BrokenItem: func() item.Stack {
			return item.Stack{}
		},
	}
}

// EncodeItem returns the Minecraft Bedrock identifier.
func (Shield) EncodeItem() (name string, meta int16) {
	return "minecraft:shield", 0
}

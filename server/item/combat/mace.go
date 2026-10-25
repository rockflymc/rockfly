package combat

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Mace represents the formidable 1.21 Tricky Trials heavy weapon capable of executing
// airborne smash attacks that scale damage quadratically with fall distance, negate fall damage,
// and propel the wielder upward with wind burst bounces.
type Mace struct {
	DensityLevel   int
	WindBurstLevel int
}

// NewMace initializes a standard unenchanted Mace.
func NewMace() Mace {
	return Mace{}
}

// AttackDamage returns 6.0 base melee attack power.
func (Mace) AttackDamage() float64 {
	return 6.0
}

// MaxCount returns 1.
func (Mace) MaxCount() int {
	return 1
}

// DurabilityInfo returns mace durability configuration (500 uses).
func (Mace) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{
		MaxDurability: 500,
		BrokenItem: func() item.Stack {
			return item.Stack{}
		},
		AttackDurability: 1,
		BreakDurability:  2,
	}
}

// IsSmashAttack checks whether a fall distance exceeds the 1.5-block trigger threshold.
func (Mace) IsSmashAttack(fallDistance float64) bool {
	if math.IsNaN(fallDistance) || math.IsInf(fallDistance, 0) {
		return false
	}
	return fallDistance > 1.5
}

// CalculateSmashDamage computes the total attack damage delivered based on fall height and Density enchantment.
func (m Mace) CalculateSmashDamage(fallDistance float64) float64 {
	base := 6.0
	if math.IsNaN(fallDistance) || math.IsInf(fallDistance, 0) || fallDistance <= 1.5 {
		return base
	}
	if fallDistance > 1024.0 {
		fallDistance = 1024.0
	}

	fall := fallDistance - 1.5
	var bonus float64

	// Canonical 1.21 damage scaling tiers:
	// Tier 1: First 3 blocks = +4.0 damage per block (max +12.0)
	tier1 := math.Min(fall, 3.0)
	bonus += tier1 * 4.0

	// Tier 2: Next 5 blocks (blocks 4 to 8) = +2.0 damage per block (max +10.0)
	if fall > 3.0 {
		tier2 := math.Min(fall-3.0, 5.0)
		bonus += tier2 * 2.0
	}

	// Tier 3: Blocks beyond 8 = +1.0 damage per block
	if fall > 8.0 {
		tier3 := fall - 8.0
		bonus += tier3 * 1.0
	}

	// Density enchantment: adds +0.5 damage per block fallen per level
	density := m.DensityLevel
	if density < 0 {
		density = 0
	} else if density > 255 {
		density = 255
	}
	if density > 0 {
		bonus += fall * 0.5 * float64(density)
	}

	return base + bonus
}

// WindBurstBounceVelocity returns the upward bounce velocity imparted on the attacker upon smash hit.
func (m Mace) WindBurstBounceVelocity() float64 {
	wb := m.WindBurstLevel
	if wb <= 0 {
		return 0.8 // Standard smash bounce
	}
	if wb > 10 {
		wb = 10
	}
	// Wind Burst I: 0.8, II: 1.2, III: 1.6
	return 0.8 + 0.4*float64(wb)
}

// ExecuteSmashAOE inflicts radial knockback and shockwave impulse to surrounding entities within 3.0 blocks.
func (Mace) ExecuteSmashAOE(tx *world.Tx, center mgl64.Vec3, primaryTarget world.Entity) {
	if tx == nil || math.IsNaN(center[0]) || math.IsNaN(center[1]) || math.IsNaN(center[2]) {
		return
	}
	radius := 3.0
	box := cube.Box(center[0]-radius, center[1]-radius, center[2]-radius, center[0]+radius, center[1]+radius, center[2]+radius)

	for e := range tx.EntitiesWithin(box) {
		if e == primaryTarget {
			continue
		}
		pos := e.Position()
		if math.IsNaN(pos[0]) || math.IsNaN(pos[1]) || math.IsNaN(pos[2]) {
			continue
		}
		diff := pos.Sub(center)
		dist := diff.Len()
		if dist <= radius && dist > 0.001 {
			factor := 1.0 - (dist / radius)
			if living, ok := e.(entity.Living); ok {
				living.KnockBack(center, factor*1.2, 0.35)
			}
		}
	}
}

// EncodeItem returns the Minecraft Bedrock identifier.
func (Mace) EncodeItem() (name string, meta int16) {
	return "minecraft:mace", 0
}

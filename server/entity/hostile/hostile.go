package hostile

import (
	"math"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Hostile serves as the common base for aggressive mobs, integrating LivingEntity
// attributes with prioritized GoalSelector and TargetSelector behavioral pipelines.
type Hostile struct {
	*entity.LivingEntity

	mu             sync.RWMutex
	target         ai.Mob
	lastAttacker   ai.Mob
	goalSelector   *ai.GoalSelector
	targetSelector *ai.GoalSelector
}

// NewHostile initializes a Hostile mob base.
func NewHostile(handle *world.EntityHandle, maxHealth float64) *Hostile {
	return &Hostile{
		LivingEntity:   entity.NewLivingEntity(handle, maxHealth),
		goalSelector:   ai.NewGoalSelector(),
		targetSelector: ai.NewGoalSelector(),
	}
}

// Target returns the mob's active combat target.
func (h *Hostile) Target() ai.Mob {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.target
}

// SetTarget updates the mob's active combat target.
func (h *Hostile) SetTarget(t ai.Mob) {
	h.mu.Lock()
	h.target = t
	h.mu.Unlock()
}

// LastAttacker returns the last entity that inflicted damage on this mob.
func (h *Hostile) LastAttacker() ai.Mob {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastAttacker
}

// SetLastAttacker updates the mob's recorded last attacker.
func (h *Hostile) SetLastAttacker(attacker ai.Mob) {
	h.mu.Lock()
	h.lastAttacker = attacker
	h.mu.Unlock()
}

// GoalSelector returns the action goal coordinator.
func (h *Hostile) GoalSelector() *ai.GoalSelector {
	return h.goalSelector
}

// TargetSelector returns the combat target acquisition coordinator.
func (h *Hostile) TargetSelector() *ai.GoalSelector {
	return h.targetSelector
}

// Hurt overrides LivingEntity.Hurt to record the attacker for retaliation.
func (h *Hostile) Hurt(damage float64, src world.DamageSource) (float64, bool) {
	dmg, vul := h.LivingEntity.Hurt(damage, src)
	if vul {
		if atk, ok := src.(entity.AttackDamageSource); ok {
			if mob, ok := atk.Attacker.(ai.Mob); ok {
				h.mu.Lock()
				h.lastAttacker = mob
				h.mu.Unlock()
			}
		} else if proj, ok := src.(entity.ProjectileDamageSource); ok {
			if mob, ok := proj.Owner.(ai.Mob); ok {
				h.mu.Lock()
				h.lastAttacker = mob
				h.mu.Unlock()
			}
		}
	}
	return dmg, vul
}

// TickHostile coordinates target selection, AI goal execution, and living entity updates.
func (h *Hostile) TickHostile(tx *world.Tx, current int64) {
	h.Tick(tx, current)

	if !h.Dead() {
		h.targetSelector.Tick()
		h.goalSelector.Tick()
	}
}

// DistanceTo returns Euclidean distance to another entity or position.
func (h *Hostile) DistanceTo(targetPos mgl64.Vec3) float64 {
	if h == nil {
		return 0
	}
	return h.Position().Sub(targetPos).Len()
}

// LookAt rotates yaw and pitch to face targetPos.
func (h *Hostile) LookAt(targetPos mgl64.Vec3) {
	if h == nil {
		return
	}
	diff := targetPos.Sub(h.Position())
	if diff.Len() < 0.001 {
		return
	}
	horizontalDist := diff[0]*diff[0] + diff[2]*diff[2]
	yaw := float64(0)
	if horizontalDist > 0.0001 {
		yaw = -mgl64.RadToDeg(math.Atan2(diff[0], diff[2]))
	}
	pitch := -mgl64.RadToDeg(math.Atan2(diff[1], math.Sqrt(horizontalDist)))
	h.SetRotation(cube.Rotation{yaw, pitch})
}

// CheckDaylightBurning evaluates if the mob should ignite from direct sunlight.
// Returns true if ignition occurred.
func CheckDaylightBurning(mob ai.Mob, l *entity.LivingEntity, tx *world.Tx) bool {
	if mob == nil || l == nil || tx == nil || tx.World() == nil || mob.Dead() {
		return false
	}
	// Check world time: daytime is [0, 12000) ticks
	timeTicks := tx.World().Time() % 24000
	if timeTicks < 0 || timeTicks >= 12000 {
		return false
	}
	pos := cube.PosFromVec3(mob.Position())
	r := tx.World().Range()
	if pos.Y() < r.Min() || pos.Y() >= r.Max() {
		return false
	}
	if tx.RainingAt(pos) {
		return false
	}
	highest := tx.HighestLightBlocker(pos.X(), pos.Z())
	if pos.Y() < highest {
		return false
	}
	if !l.Armour().Helmet().Empty() {
		return false
	}
	l.SetOnFire(8 * time.Second)
	return true
}

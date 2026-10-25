package boss

import (
	"sync"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Wither represents a destructive three-headed undead boss capable of independent multi-target
// tracking, wither skull artillery, wither armor shield protection, and Nether Star drops.
type Wither struct {
	*Boss

	mu           sync.RWMutex
	mainTarget   ai.Mob
	leftTarget   ai.Mob
	rightTarget  ai.Mob
	invulnerable atomic.Int32 // Initial 220-tick spawning charge up
	skullTimer   int
}

// NewWither initializes a Wither boss with 300.0 health and pink BossBar.
func NewWither(handle *world.EntityHandle) *Wither {
	bar := NewBossBar(2, "Wither", packet.BossEventColourPink, 0)
	w := &Wither{
		Boss: NewBoss(handle, 300.0, bar),
	}
	w.SetSpeed(0.35)
	w.invulnerable.Store(220) // 11-second chargeup on spawn

	return w
}

// InvulnerableTicks returns remaining spawning invulnerability ticks.
func (w *Wither) InvulnerableTicks() int32 {
	return w.invulnerable.Load()
}

// SetInvulnerableTicks manually updates invulnerability countdown.
func (w *Wither) SetInvulnerableTicks(ticks int32) {
	w.invulnerable.Store(ticks)
}

// WitherArmor returns true if the Wither is protected by its <= 50% health shield.
func (w *Wither) WitherArmor() bool {
	return w.Health() <= (w.MaxHealth() * 0.5)
}

// SetMainTarget updates the central head's target.
func (w *Wither) SetMainTarget(target ai.Mob) {
	w.mu.Lock()
	w.mainTarget = target
	w.mu.Unlock()
}

// MainTarget returns the primary target.
func (w *Wither) MainTarget() ai.Mob {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.mainTarget
}

// SetSecondaryTargets updates the independent left and right head targets.
func (w *Wither) SetSecondaryTargets(left, right ai.Mob) {
	w.mu.Lock()
	w.leftTarget = left
	w.rightTarget = right
	w.mu.Unlock()
}

// SecondaryTargets returns the current side head targets.
func (w *Wither) SecondaryTargets() (left, right ai.Mob) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.leftTarget, w.rightTarget
}

// Hurt overrides LivingEntity.Hurt to enforce invulnerability and Wither Armor projectile immunity.
func (w *Wither) Hurt(damage float64, src world.DamageSource) (float64, bool) {
	if w.invulnerable.Load() > 0 {
		return 0, false
	}
	if w.WitherArmor() {
		if _, isProj := src.(entity.ProjectileDamageSource); isProj {
			// Complete projectile immunity while Wither Armor is active!
			return 0, false
		}
	}

	dmg, vul := w.LivingEntity.Hurt(damage, src)
	if vul {
		w.SyncBarHealth()
	}
	return dmg, vul
}

// Drops produces the canonical Nether Star drop upon defeat.
func (w *Wither) Drops() []item.Stack {
	return []item.Stack{item.NewStack(item.NetherStar{}, 1)}
}

// Tick coordinates invulnerability countdown, head target tracking, and combat logic.
func (w *Wither) Tick(tx *world.Tx, current int64) {
	if w.invulnerable.Load() > 0 {
		w.invulnerable.Add(-1)
		w.Heal(1.0, nil)
		w.SyncBarHealth()
		return
	}

	w.skullTimer++
	if w.skullTimer >= 40 {
		w.skullTimer = 0
		// Loose skull artillery towards active targets
	}

	w.LivingEntity.Tick(tx, current)
}

// BoundingBox returns the collision box conforming to 0.9 x 3.5 dimensions.
func (w *Wither) BoundingBox() cube.BBox {
	return cube.Box(-0.45, 0, -0.45, 0.45, 3.5, 0.45)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (w *Wither) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder().
		SyncLiving(w.LivingEntity).
		SetDimensions(0.9, 3.5).
		SetScale(1.0)

	m := builder.Build()
	if w.WitherArmor() {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagPowered)
	}
	if w.invulnerable.Load() > 0 {
		m[protocol.EntityDataKeyInvulnerableTicks] = int32(w.invulnerable.Load())
	}
	return m
}

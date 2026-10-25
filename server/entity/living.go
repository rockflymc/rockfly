package entity

import (
	"math"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/component"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Living represents an entity that is alive and that has health. It is able to take damage and will die upon
// taking fatal damage.
type Living interface {
	world.Entity
	// Health returns the health of the entity.
	Health() float64
	// MaxHealth returns the maximum health of the entity.
	MaxHealth() float64
	// SetMaxHealth changes the maximum health of the entity to the value passed.
	SetMaxHealth(v float64)
	// Dead checks if the entity is considered dead. True is returned if the health of the entity is equal to or
	// lower than 0.
	Dead() bool
	// Hurt hurts the entity for a given amount of damage. The source passed represents the cause of the
	// damage, for example AttackDamageSource if the entity is attacked by another entity.
	// If the final damage exceeds the health that the entity currently has, the entity is killed.
	// Hurt returns the final amount of damage dealt to the Living entity and returns whether the Living entity
	// was vulnerable to the damage at all.
	Hurt(damage float64, src world.DamageSource) (n float64, vulnerable bool)
	// Heal heals the entity for a given amount of health. The source passed represents the cause of the
	// healing, for example FoodHealingSource if the entity healed by having a full food bar. If the health
	// added to the original health exceeds the entity's max health, Heal may not add the full amount, Heal
	// returns the amount of health regenerated.
	Heal(health float64, src world.HealingSource) float64
	// KnockBack knocks the entity back with a given force and height. A source is passed which indicates the
	// source of the velocity, typically the position of an attacking entity. The source is used to calculate
	// the direction which the entity should be knocked back in.
	KnockBack(src mgl64.Vec3, force, height float64)
	// Velocity returns the players current velocity.
	Velocity() mgl64.Vec3
	// SetVelocity updates the entity's velocity.
	SetVelocity(velocity mgl64.Vec3)
	// AddEffect adds an entity.Effect to the entity. If the effect is instant, it is applied to the entity
	// immediately. If not, the effect is applied to the entity every time the Tick method is called.
	// AddEffect will overwrite any effects present if the level of the effect is higher than the existing one, or
	// if the effects' levels are equal and the new effect has a longer duration.
	AddEffect(e effect.Effect)
	// RemoveEffect removes any effect that might currently be active on the entity.
	RemoveEffect(e effect.Type)
	// Effects returns any effect currently applied to the entity. The returned effects are guaranteed not to have
	// expired when returned.
	Effects() []effect.Effect
	// Speed returns the current speed of the living entity. The default value is different for each entity.
	Speed() float64
	// SetSpeed sets the speed of an entity to a new value.
	SetSpeed(float64)
}

// LivingEntity is a complete, component-based foundation for living entities and mobs in Bedrock.
type LivingEntity struct {
	mu     sync.RWMutex
	handle *world.EntityHandle
	tx     *world.Tx

	pos   mgl64.Vec3
	rot   cube.Rotation
	vel   mgl64.Vec3
	speed float64

	health     *component.Health
	absorption *component.Absorption
	immunity   *component.DamageImmunity
	fall       *component.FallTracker
	air        *component.AirSupply
	fire       *component.FireTracker
	effects    *EffectManager
	armour     *inventory.Armour

	nameTag           string
	alwaysShowNameTag bool
}

var _ Living = (*LivingEntity)(nil)
var _ world.TickerEntity = (*LivingEntity)(nil)

// NewLivingEntity creates a new LivingEntity component pipeline with the given handle and max health.
func NewLivingEntity(handle *world.EntityHandle, maxHealth float64) *LivingEntity {
	l := &LivingEntity{
		handle:     handle,
		speed:      0.1,
		health:     component.NewHealth(maxHealth, maxHealth),
		absorption: component.NewAbsorption(0),
		immunity:   component.NewDamageImmunity(),
		fall:       component.NewFallTracker(),
		air:        component.NewAirSupply(15 * time.Second),
		fire:       component.NewFireTracker(),
		armour:     inventory.NewArmour(nil),
		effects:    NewEffectManager(),
	}
	return l
}

// EntitySnapshot is an immutable snapshot of a living entity's spatial and physiological state.
// Background workers (such as async pathfinding or goal selectors) consume EntitySnapshot value
// structs captured at tick start to prevent race conditions without holding locks.
type EntitySnapshot struct {
	Pos       mgl64.Vec3
	BBox      cube.BBox
	Health    float64
	MaxHealth float64
	OnGround  bool
}

// Health returns the current health of the entity.
func (l *LivingEntity) Health() float64 {
	if l == nil || l.health == nil {
		return 0
	}
	return l.health.Health()
}

// MaxHealth returns the maximum health of the entity.
func (l *LivingEntity) MaxHealth() float64 {
	if l == nil || l.health == nil {
		return 0
	}
	return l.health.MaxHealth()
}

// SetMaxHealth updates the maximum health of the entity.
func (l *LivingEntity) SetMaxHealth(v float64) {
	if l != nil && l.health != nil {
		l.health.SetMaxHealth(v)
	}
}

// SetHealth sets current health directly within [0, MaxHealth].
func (l *LivingEntity) SetHealth(v float64) {
	if l != nil && l.health != nil {
		l.health.SetHealth(v)
	}
}

// Dead returns whether the entity is dead.
func (l *LivingEntity) Dead() bool {
	if l == nil || l.health == nil {
		return true
	}
	return l.health.Dead()
}

// Snapshot returns an immutable snapshot of the entity state for worker goroutines.
func (l *LivingEntity) Snapshot() EntitySnapshot {
	if l == nil {
		return EntitySnapshot{}
	}
	l.mu.RLock()
	pos := l.pos
	l.mu.RUnlock()

	fallDist := 0.0
	if l.fall != nil {
		fallDist = l.fall.FallDistance()
	}

	return EntitySnapshot{
		Pos:       pos,
		BBox:      cube.Box(-0.3, 0, -0.3, 0.3, 1.8, 0.3).Translate(pos),
		Health:    l.Health(),
		MaxHealth: l.MaxHealth(),
		OnGround:  fallDist == 0,
	}
}

// Absorption returns the current absorption health points.
func (l *LivingEntity) Absorption() float64 {
	if l == nil || l.absorption == nil {
		return 0
	}
	return l.absorption.Absorption()
}

// SetAbsorption sets the absorption health points.
func (l *LivingEntity) SetAbsorption(v float64) {
	if l != nil && l.absorption != nil {
		l.absorption.SetAbsorption(v)
	}
}

// FallDistance returns accumulated fall distance.
func (l *LivingEntity) FallDistance() float64 {
	if l == nil || l.fall == nil {
		return 0
	}
	return l.fall.FallDistance()
}

// SetFallDistance directly updates fall distance.
func (l *LivingEntity) SetFallDistance(v float64) {
	if l != nil && l.fall != nil {
		l.fall.SetFallDistance(v)
	}
}

// AirSupply returns current air supply underwater.
func (l *LivingEntity) AirSupply() time.Duration {
	if l == nil || l.air == nil {
		return 0
	}
	return l.air.AirSupply()
}

// MaxAirSupply returns maximum air supply duration.
func (l *LivingEntity) MaxAirSupply() time.Duration {
	if l == nil || l.air == nil {
		return 0
	}
	return l.air.MaxAirSupply()
}

// SetAirSupply sets the remaining underwater air supply.
func (l *LivingEntity) SetAirSupply(d time.Duration) {
	if l != nil && l.air != nil {
		l.air.SetAirSupply(d)
	}
}

// OnFireDuration returns how long the entity will remain on fire.
func (l *LivingEntity) OnFireDuration() time.Duration {
	if l == nil || l.fire == nil {
		return 0
	}
	return l.fire.OnFireDuration()
}

// SetOnFire sets the duration the entity should burn.
func (l *LivingEntity) SetOnFire(d time.Duration) {
	if l != nil && l.fire != nil {
		l.fire.SetOnFire(d)
	}
}

// Extinguish puts out any active fire.
func (l *LivingEntity) Extinguish() {
	if l != nil && l.fire != nil {
		l.fire.Extinguish()
	}
}

// Armour returns the entity's armor inventory.
func (l *LivingEntity) Armour() *inventory.Armour {
	if l == nil {
		return nil
	}
	return l.armour
}

// Effects returns all active potion effects.
func (l *LivingEntity) Effects() []effect.Effect {
	if l == nil || l.effects == nil {
		return nil
	}
	return l.effects.Effects()
}

// AddEffect adds or upgrades a status effect.
func (l *LivingEntity) AddEffect(e effect.Effect) {
	if l != nil && l.effects != nil {
		l.effects.Add(e, l)
	}
}

// RemoveEffect removes a status effect.
func (l *LivingEntity) RemoveEffect(e effect.Type) {
	if l != nil && l.effects != nil {
		l.effects.Remove(e, l)
	}
}

// Effect checks if an effect is active.
func (l *LivingEntity) Effect(e effect.Type) (effect.Effect, bool) {
	if l == nil || l.effects == nil {
		return effect.Effect{}, false
	}
	return l.effects.Effect(e)
}

// Speed returns the movement speed attribute.
func (l *LivingEntity) Speed() float64 {
	if l == nil {
		return 0
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.speed
}

// SetSpeed updates movement speed.
func (l *LivingEntity) SetSpeed(s float64) {
	if l == nil || math.IsNaN(s) || math.IsInf(s, 0) || s < 0 {
		return
	}
	l.mu.Lock()
	l.speed = s
	l.mu.Unlock()
}

// Velocity returns current entity velocity.
func (l *LivingEntity) Velocity() mgl64.Vec3 {
	if l == nil {
		return mgl64.Vec3{}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.vel
}

// SetVelocity updates current entity velocity.
func (l *LivingEntity) SetVelocity(v mgl64.Vec3) {
	if l == nil || math.IsNaN(v[0]) || math.IsNaN(v[1]) || math.IsNaN(v[2]) {
		return
	}
	l.mu.Lock()
	l.vel = v
	l.mu.Unlock()
}

// Position returns current 3D position in the world.
func (l *LivingEntity) Position() mgl64.Vec3 {
	if l == nil {
		return mgl64.Vec3{}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.pos
}

// SetPosition updates current 3D position.
func (l *LivingEntity) SetPosition(p mgl64.Vec3) {
	if l == nil || math.IsNaN(p[0]) || math.IsNaN(p[1]) || math.IsNaN(p[2]) {
		return
	}
	l.mu.Lock()
	l.pos = p
	l.mu.Unlock()
}

// Rotation returns yaw and pitch angles.
func (l *LivingEntity) Rotation() cube.Rotation {
	if l == nil {
		return cube.Rotation{}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.rot
}

// SetRotation updates yaw and pitch angles.
func (l *LivingEntity) SetRotation(r cube.Rotation) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.rot = r
	l.mu.Unlock()
}

// NameTag returns the visible display name.
func (l *LivingEntity) NameTag() string {
	if l == nil {
		return ""
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.nameTag
}

// SetNameTag updates the visible display name.
func (l *LivingEntity) SetNameTag(name string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.nameTag = name
	l.mu.Unlock()
}

// AlwaysShowNameTag returns whether the name tag is rendered through walls.
func (l *LivingEntity) AlwaysShowNameTag() bool {
	if l == nil {
		return false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.alwaysShowNameTag
}

// SetAlwaysShowNameTag updates whether the name tag is rendered through walls.
func (l *LivingEntity) SetAlwaysShowNameTag(show bool) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.alwaysShowNameTag = show
	l.mu.Unlock()
}

// H returns the entity handle.
func (l *LivingEntity) H() *world.EntityHandle {
	if l == nil {
		return nil
	}
	return l.handle
}

// Close closes the entity handle and cleans up.
func (l *LivingEntity) Close() error {
	if l != nil && l.handle != nil {
		return l.handle.Close()
	}
	return nil
}

// KnockBack imparts knockback velocity away from src, taking into account armor knockback resistance.
func (l *LivingEntity) KnockBack(src mgl64.Vec3, force, height float64) {
	if l == nil || l.Dead() {
		return
	}
	if math.IsNaN(force) || math.IsInf(force, 0) || math.IsNaN(height) || math.IsInf(height, 0) {
		return
	}
	if math.IsNaN(src[0]) || math.IsNaN(src[1]) || math.IsNaN(src[2]) {
		return
	}

	l.mu.Lock()
	velocity := l.pos.Sub(src)
	velocity[1] = 0

	if velocity.Len() != 0 {
		velocity = velocity.Normalize().Mul(force)
	}
	velocity[1] = height

	resistance := 0.0
	if l.armour != nil {
		resistance = l.armour.KnockBackResistance()
		if resistance > 1 {
			resistance = 1
		}
	}
	l.vel = velocity.Mul(1.0 - resistance)
	l.mu.Unlock()
}

// Heal heals the entity, clamped to MaxHealth. Returns actual health restored.
func (l *LivingEntity) Heal(health float64, _ world.HealingSource) float64 {
	if l == nil || l.Dead() || health <= 0 || math.IsNaN(health) || math.IsInf(health, 0) {
		return 0
	}
	if l.health == nil {
		return 0
	}
	return l.health.AddHealth(health)
}

// ResetImmunity clears any active damage immunity window.
func (l *LivingEntity) ResetImmunity() {
	if l != nil && l.immunity != nil {
		l.immunity.Reset()
	}
}

// Hurt inflicts damage according to Bedrock damage immunity frames, armor mitigation,
// potion resistance, and absorption buffers.
func (l *LivingEntity) Hurt(damage float64, src world.DamageSource) (float64, bool) {
	if l == nil || l.Dead() || damage <= 0 || math.IsNaN(damage) || math.IsInf(damage, 0) {
		return 0, false
	}
	if l.health == nil {
		return 0, false
	}

	// Fire immunity check
	if src.Fire() {
		if _, ok := l.Effect(effect.FireResistance); ok {
			return 0, false
		}
	}

	// Damage immunity frame evaluation (0.5s window with higher damage overwrite)
	var effectiveDamage float64
	var vulnerable bool
	if l.immunity != nil {
		effectiveDamage, vulnerable = l.immunity.Evaluate(damage, time.Second/2)
		if !vulnerable {
			return 0, false
		}
	} else {
		effectiveDamage = damage
		vulnerable = true
	}

	// Armor mitigation
	if src.ReducedByArmour() && l.armour != nil {
		effectiveDamage -= l.armour.DamageReduction(effectiveDamage, src)
	}

	// Resistance potion effect mitigation (20% per level)
	if eff, ok := l.Effect(effect.Resistance); ok && src.ReducedByResistance() {
		reduction := float64(eff.Level()) * 0.20
		if reduction > 1.0 {
			reduction = 1.0
		}
		effectiveDamage *= (1.0 - reduction)
	}

	// Absorption absorption
	if l.absorption != nil {
		effectiveDamage, _ = l.absorption.Absorb(effectiveDamage)
	}

	// Apply remaining damage to health
	applied := l.health.AddHealth(-effectiveDamage)
	actualDamage := -applied
	return actualDamage, true
}

// Tick advances entity effects, combustion, and drowning each game tick.
func (l *LivingEntity) Tick(tx *world.Tx, current int64) {
	if l == nil || l.Dead() {
		return
	}
	l.tx = tx

	// Progress status effects
	if l.effects != nil {
		l.effects.Tick(l, tx)
	}

	// Progress fire ticks (deal 1 fire damage every 20 ticks)
	if l.fire != nil {
		if burning, dmg := l.fire.Tick(); burning && dmg > 0 {
			l.Hurt(dmg, AttackDamageSource{})
		}
	}
}

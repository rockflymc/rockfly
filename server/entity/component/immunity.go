package component

import (
	"math"
	"sync"
	"time"
)

// DamageImmunity tracks Bedrock-accurate damage immunity frames (0.5s window).
// When an entity is struck, subsequent hits within the immunity frame only deal
// damage if the new hit is stronger than the original hit, in which case only
// the delta (incoming - lastDamage) is inflicted.
type DamageImmunity struct {
	mu          sync.Mutex
	immuneUntil time.Time
	lastDamage  float64
}

// NewDamageImmunity creates a new DamageImmunity component.
func NewDamageImmunity() *DamageImmunity {
	return &DamageImmunity{}
}

// Evaluate evaluates an incoming hit of amount damage. If the entity is currently
// within its immunity frame, damage is only applied if damage > lastDamage.
// If vulnerable, the effective damage to deal is returned along with vulnerable=true.
func (d *DamageImmunity) Evaluate(damage float64, duration time.Duration) (effectiveDamage float64, vulnerable bool) {
	if d == nil {
		return damage, true
	}
	if damage <= 0 || math.IsNaN(damage) || math.IsInf(damage, 0) {
		return 0, false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if now.Before(d.immuneUntil) {
		// Inside immunity frame: only inflict higher damage overwrite
		if damage > d.lastDamage {
			effectiveDamage = damage - d.lastDamage
			d.lastDamage = damage
			return effectiveDamage, true
		}
		// Incoming damage is less than or equal to previous hit: full immunity
		return 0, false
	}

	// Outside immunity window: full damage dealt and new immunity window established
	d.immuneUntil = now.Add(duration)
	d.lastDamage = damage
	return damage, true
}

// Reset clears the immunity frame immediately.
func (d *DamageImmunity) Reset() {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.immuneUntil = time.Time{}
	d.lastDamage = 0
	d.mu.Unlock()
}

// Immune returns whether the entity is currently within an immunity window.
func (d *DamageImmunity) Immune() bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return time.Now().Before(d.immuneUntil)
}

// LastDamage returns the damage of the last hit within the current immunity frame.
func (d *DamageImmunity) LastDamage() float64 {
	if d == nil {
		return 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastDamage
}

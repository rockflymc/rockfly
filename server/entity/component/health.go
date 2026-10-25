package component

import (
	"math"
	"sync/atomic"
)

// Health tracks current and maximum health of a living entity.
type Health struct {
	current atomic.Uint64
	max     atomic.Uint64
}

// NewHealth initializes a Health component with initial and maximum health.
func NewHealth(health, maxHealth float64) *Health {
	if maxHealth <= 0 {
		maxHealth = 1
	}
	if health > maxHealth {
		health = maxHealth
	}
	if health < 0 {
		health = 0
	}
	h := &Health{}
	h.current.Store(math.Float64bits(health))
	h.max.Store(math.Float64bits(maxHealth))
	return h
}

// Health returns the current health value.
func (h *Health) Health() float64 {
	return math.Float64frombits(h.current.Load())
}

// MaxHealth returns the maximum health value.
func (h *Health) MaxHealth() float64 {
	return math.Float64frombits(h.max.Load())
}

// SetMaxHealth updates the maximum health. If the new max is less than the current
// health, current health is clamped to max.
func (h *Health) SetMaxHealth(maxHealth float64) {
	if maxHealth <= 0 {
		maxHealth = 1
	}
	h.max.Store(math.Float64bits(maxHealth))
	for {
		curBits := h.current.Load()
		cur := math.Float64frombits(curBits)
		if cur <= maxHealth {
			break
		}
		if h.current.CompareAndSwap(curBits, math.Float64bits(maxHealth)) {
			break
		}
	}
}

// SetHealth sets current health directly, clamped between 0 and MaxHealth.
func (h *Health) SetHealth(health float64) {
	maxVal := h.MaxHealth()
	health = max(0, min(health, maxVal))
	h.current.Store(math.Float64bits(health))
}

// AddHealth adds health (or removes if negative), clamping between 0 and MaxHealth.
// Returns the actual delta applied.
func (h *Health) AddHealth(delta float64) float64 {
	maxVal := h.MaxHealth()
	for {
		curBits := h.current.Load()
		cur := math.Float64frombits(curBits)
		next := max(0, min(cur+delta, maxVal))
		if h.current.CompareAndSwap(curBits, math.Float64bits(next)) {
			return next - cur
		}
	}
}

// Dead returns whether health is 0 or below.
func (h *Health) Dead() bool {
	return h.Health() <= 0
}

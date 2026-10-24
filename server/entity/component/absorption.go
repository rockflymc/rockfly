package component

import (
	"math"
	"sync/atomic"
)

// Absorption tracks extra absorption health points (yellow hearts).
type Absorption struct {
	value atomic.Uint64
}

// NewAbsorption creates an Absorption component with initial absorption amount.
func NewAbsorption(initial float64) *Absorption {
	if initial < 0 {
		initial = 0
	}
	a := &Absorption{}
	a.value.Store(math.Float64bits(initial))
	return a
}

// Absorption returns the current absorption health points.
func (a *Absorption) Absorption() float64 {
	return math.Float64frombits(a.value.Load())
}

// SetAbsorption sets the absorption health value.
func (a *Absorption) SetAbsorption(val float64) {
	if val < 0 {
		val = 0
	}
	a.value.Store(math.Float64bits(val))
}

// Absorb takes an incoming damage amount and absorbs as much as possible using
// the current absorption buffer. It returns the remaining damage and the amount absorbed.
func (a *Absorption) Absorb(damage float64) (damageLeft, absorbed float64) {
	if damage <= 0 {
		return 0, 0
	}
	for {
		curBits := a.value.Load()
		cur := math.Float64frombits(curBits)
		if cur <= 0 {
			return damage, 0
		}
		if cur >= damage {
			next := cur - damage
			if a.value.CompareAndSwap(curBits, math.Float64bits(next)) {
				return 0, damage
			}
		} else {
			// cur < damage
			if a.value.CompareAndSwap(curBits, 0) {
				return damage - cur, cur
			}
		}
	}
}

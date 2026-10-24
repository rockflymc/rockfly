package component

import (
	"math"
	"sync/atomic"
)

// FallTracker tracks kinetic fall distance for living entities.
type FallTracker struct {
	distance atomic.Uint64
}

// NewFallTracker creates a new FallTracker component.
func NewFallTracker() *FallTracker {
	return &FallTracker{}
}

// FallDistance returns the current accumulated fall distance in blocks.
func (f *FallTracker) FallDistance() float64 {
	return math.Float64frombits(f.distance.Load())
}

// SetFallDistance directly sets the accumulated fall distance.
func (f *FallTracker) SetFallDistance(dist float64) {
	if dist < 0 {
		dist = 0
	}
	f.distance.Store(math.Float64bits(dist))
}

// Accumulate adds vertical descent to the fall distance. If climbing or rising (deltaY > 0),
// fall distance is reset to 0.
func (f *FallTracker) Accumulate(deltaY float64) {
	if deltaY >= 0 {
		f.distance.Store(0)
		return
	}
	for {
		curBits := f.distance.Load()
		cur := math.Float64frombits(curBits)
		next := cur - deltaY
		if f.distance.CompareAndSwap(curBits, math.Float64bits(next)) {
			break
		}
	}
}

// Impact calculates kinetic fall damage upon landing on solid ground and resets the fall distance.
// Vanilla formula: max(0, fallDistance - 3.0).
func (f *FallTracker) Impact(featherFallingLevel int) float64 {
	for {
		curBits := f.distance.Load()
		cur := math.Float64frombits(curBits)
		if f.distance.CompareAndSwap(curBits, 0) {
			if cur <= 3.0 {
				return 0
			}
			damage := cur - 3.0
			if featherFallingLevel > 0 {
				reduction := float64(featherFallingLevel) * 0.12
				if reduction > 0.80 {
					reduction = 0.80
				}
				damage *= (1.0 - reduction)
			}
			return damage
		}
	}
}

// Reset clears the accumulated fall distance.
func (f *FallTracker) Reset() {
	f.distance.Store(0)
}

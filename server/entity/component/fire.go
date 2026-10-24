package component

import (
	"sync/atomic"
	"time"
)

// FireTracker tracks active combustion and burning duration.
type FireTracker struct {
	fireTicks atomic.Int64
	burnTimer atomic.Int32
}

// NewFireTracker creates an unlit FireTracker.
func NewFireTracker() *FireTracker {
	return &FireTracker{}
}

// OnFireDuration returns the remaining duration the entity is on fire.
func (f *FireTracker) OnFireDuration() time.Duration {
	ticks := f.fireTicks.Load()
	if ticks <= 0 {
		return 0
	}
	return time.Duration(ticks) * 50 * time.Millisecond
}

// SetOnFire sets the duration the entity should burn.
func (f *FireTracker) SetOnFire(d time.Duration) {
	ticks := int64(d.Milliseconds() / 50)
	if ticks < 0 {
		ticks = 0
	}
	f.fireTicks.Store(ticks)
}

// Extinguish immediately stops combustion.
func (f *FireTracker) Extinguish() {
	f.fireTicks.Store(0)
	f.burnTimer.Store(0)
}

// Tick decrements combustion ticks and inflicts 1.0 fire damage every 20 ticks (1 second).
func (f *FireTracker) Tick() (burning bool, damage float64) {
	rem := f.fireTicks.Add(-1)
	if rem < 0 {
		f.fireTicks.Store(0)
		f.burnTimer.Store(0)
		return false, 0
	}

	timer := f.burnTimer.Add(1)
	if timer >= 20 {
		f.burnTimer.Store(0)
		return true, 1.0
	}
	return true, 0
}

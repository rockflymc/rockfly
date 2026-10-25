package component

import (
	"sync/atomic"
	"time"
)

// AirSupply tracks oxygen levels when submerged underwater.
type AirSupply struct {
	airTicks    atomic.Int32
	maxAirTicks atomic.Int32
	drownTimer  atomic.Int32
}

// NewAirSupply creates an AirSupply component initialized to maxSupply.
func NewAirSupply(maxSupply time.Duration) *AirSupply {
	ticks := int32(maxSupply.Milliseconds() / 50)
	if ticks <= 0 {
		ticks = 300 // default 15s (300 ticks)
	}
	a := &AirSupply{}
	a.airTicks.Store(ticks)
	a.maxAirTicks.Store(ticks)
	return a
}

// AirSupply returns the remaining air duration.
func (a *AirSupply) AirSupply() time.Duration {
	if a == nil {
		return 0
	}
	return time.Duration(a.airTicks.Load()) * 50 * time.Millisecond
}

// MaxAirSupply returns the maximum air duration.
func (a *AirSupply) MaxAirSupply() time.Duration {
	if a == nil {
		return 0
	}
	return time.Duration(a.maxAirTicks.Load()) * 50 * time.Millisecond
}

// SetAirSupply sets the remaining air duration.
func (a *AirSupply) SetAirSupply(d time.Duration) {
	if a == nil {
		return
	}
	ticks := int32(d.Milliseconds() / 50)
	maxVal := a.maxAirTicks.Load()
	a.airTicks.Store(max(0, min(ticks, maxVal)))
}

// TickSubmerged handles submerged oxygen consumption. When air runs out,
// drowning damage (1.0) is returned every 20 ticks.
func (a *AirSupply) TickSubmerged() (drowning bool, damage float64) {
	if a == nil {
		return false, 0
	}
	rem := a.airTicks.Add(-1)
	if rem > 0 {
		return false, 0
	}

	// Air exhausted: accumulate drown ticks
	timer := a.drownTimer.Add(1)
	if timer >= 20 {
		a.drownTimer.Store(0)
		return true, 1.0
	}
	return true, 0
}

// Replenish resets air supply back to full.
func (a *AirSupply) Replenish() {
	if a == nil {
		return
	}
	a.airTicks.Store(a.maxAirTicks.Load())
	a.drownTimer.Store(0)
}

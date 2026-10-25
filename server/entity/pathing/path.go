package pathing

import (
	"github.com/go-gl/mathgl/mgl64"
)

// Path represents an active sequence of 3D continuous waypoints for an entity to traverse.
type Path struct {
	waypoints []mgl64.Vec3
	index     int
}

// NewPath creates a new traversable Path from 3D coordinates.
func NewPath(waypoints []mgl64.Vec3) *Path {
	return &Path{waypoints: waypoints, index: 0}
}

// Current returns the current target waypoint. If path has ended, ok is false.
func (p *Path) Current() (mgl64.Vec3, bool) {
	if p == nil || p.index >= len(p.waypoints) {
		return mgl64.Vec3{}, false
	}
	return p.waypoints[p.index], true
}

// Advance advances the path index to the next waypoint.
func (p *Path) Advance() {
	if p != nil && p.index < len(p.waypoints) {
		p.index++
	}
}

// Next advances and returns the next waypoint.
func (p *Path) Next() (mgl64.Vec3, bool) {
	p.Advance()
	return p.Current()
}

// AtEnd returns true if all waypoints have been reached.
func (p *Path) AtEnd() bool {
	return p == nil || p.index >= len(p.waypoints)
}

// Length returns the total number of waypoints.
func (p *Path) Length() int {
	if p == nil {
		return 0
	}
	return len(p.waypoints)
}

// Waypoints returns a copy of the path waypoints.
func (p *Path) Waypoints() []mgl64.Vec3 {
	if p == nil {
		return nil
	}
	res := make([]mgl64.Vec3, len(p.waypoints))
	copy(res, p.waypoints)
	return res
}

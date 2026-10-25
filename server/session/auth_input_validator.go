package session

import (
	"fmt"
	"math"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

// MovementValidationConfig defines safety thresholds for server-authoritative movement validation.
type MovementValidationConfig struct {
	// Enabled toggles whether movement packets are validated.
	Enabled bool
	// MaxHorizontalSpeed is the maximum allowable horizontal distance per tick in blocks.
	MaxHorizontalSpeed float64
	// MaxVerticalSpeed is the maximum allowable upward vertical distance per tick in blocks.
	MaxVerticalSpeed float64
	// MaxGlidingSpeed is the maximum allowable horizontal speed when gliding with elytra.
	MaxGlidingSpeed float64
	// CheckPhase toggles phase / no-clip checks through solid blocks.
	CheckPhase bool
	// CheckFlight toggles mid-air flight and hovering verification.
	CheckFlight bool
	// MaxAirTicks is the maximum number of consecutive airborne ticks before flight is flagged.
	MaxAirTicks int
}

// DefaultMovementValidationConfig provides balanced production thresholds preventing speed, fly,
// and phase hacks while tolerating high-ping network latency and vanilla sprint-jumping.
func DefaultMovementValidationConfig() MovementValidationConfig {
	return MovementValidationConfig{
		Enabled:            true,
		MaxHorizontalSpeed: 2.5,  // Normal sprint-jump is ~0.65; allows speed potions & ice runways
		MaxVerticalSpeed:   3.0,  // Allows jump boost and wind bursts
		MaxGlidingSpeed:    5.0,  // Allows fast elytra rocket boosts
		CheckPhase:         true, // Blocks phase/no-clip through walls
		CheckFlight:        true, // Blocks hovering and mid-air ascent
		MaxAirTicks:        80,   // ~4 seconds of air time before flagging fly hacks
	}
}

// MovementValidator provides server-authoritative physics, speed, flight, and phase verification.
type MovementValidator struct {
	mu       sync.Mutex
	conf     MovementValidationConfig
	airTicks map[uuid.UUID]int
}

// NewMovementValidator creates a new MovementValidator with the specified configuration.
func NewMovementValidator(conf MovementValidationConfig) *MovementValidator {
	return &MovementValidator{
		conf:     conf,
		airTicks: make(map[uuid.UUID]int),
	}
}

// Validate checks whether movement from oldPos to newPos is physically permissible.
// Returns valid = true if movement passes checks, or valid = false with a reason describing the violation.
func (v *MovementValidator) Validate(c Controllable, tx *world.Tx, oldPos, newPos mgl64.Vec3) (bool, string) {
	if v == nil || !v.conf.Enabled {
		return true, ""
	}
	if c == nil {
		return true, ""
	}

	// 0. Coordinate Integrity Check: Reject NaN / Infinite positions immediately
	if math.IsNaN(newPos[0]) || math.IsNaN(newPos[1]) || math.IsNaN(newPos[2]) ||
		math.IsInf(newPos[0], 0) || math.IsInf(newPos[1], 0) || math.IsInf(newPos[2], 0) ||
		math.IsNaN(oldPos[0]) || math.IsNaN(oldPos[1]) || math.IsNaN(oldPos[2]) ||
		math.IsInf(oldPos[0], 0) || math.IsInf(oldPos[1], 0) || math.IsInf(oldPos[2], 0) {
		return false, "invalid coordinates: position contains NaN or Inf"
	}

	// 1. Spectator mode bypasses all collision and speed restrictions
	if gm := c.GameMode(); gm == world.GameModeSpectator {
		return true, ""
	}

	delta := newPos.Sub(oldPos)
	dx, dy, dz := delta[0], delta[1], delta[2]
	horizontalDist := math.Sqrt(dx*dx + dz*dz)

	// 2. Speed Check
	maxHorizontal := v.conf.MaxHorizontalSpeed
	if c.Gliding() {
		maxHorizontal = v.conf.MaxGlidingSpeed
	} else if c.Flying() {
		maxHorizontal = math.Max(maxHorizontal, c.FlightSpeed()*1.5)
	}

	if horizontalDist > maxHorizontal {
		return false, fmt.Sprintf("speed violation: moved %.2f blocks horizontally (max %.2f)", horizontalDist, maxHorizontal)
	}

	if dy > v.conf.MaxVerticalSpeed && !c.Flying() && !c.Gliding() {
		return false, fmt.Sprintf("vertical speed violation: moved %.2f blocks upward (max %.2f)", dy, v.conf.MaxVerticalSpeed)
	}

	// 3. Phase / No-Clip Check
	if v.conf.CheckPhase && tx != nil {
		targetBlockPos := cube.PosFromVec3(newPos)
		targetBlock := tx.Block(targetBlockPos)
		if targetBlock.Model().FaceSolid(targetBlockPos, cube.FaceUp, tx) &&
			targetBlock.Model().FaceSolid(targetBlockPos, cube.FaceDown, tx) {
			// Check head block as well
			headPos := cube.PosFromVec3(newPos.Add(mgl64.Vec3{0, 1.5, 0}))
			headBlock := tx.Block(headPos)
			if headBlock.Model().FaceSolid(headPos, cube.FaceUp, tx) {
				return false, fmt.Sprintf("phase violation: target position %v is inside solid %T", targetBlockPos, targetBlock)
			}
		}
	}

	// 4. Flight / Hover Check
	if v.conf.CheckFlight && !c.Flying() && !c.Gliding() && !c.Swimming() {
		id := c.UUID()
		v.mu.Lock()
		defer v.mu.Unlock()

		if v.airTicks == nil {
			v.airTicks = make(map[uuid.UUID]int)
		}

		// Check if player is standing on solid ground or liquid
		belowPos := cube.PosFromVec3(newPos.Sub(mgl64.Vec3{0, 0.5, 0}))
		var onGround bool
		if tx != nil {
			belowBlock := tx.Block(belowPos)
			onGround = belowBlock.Model().FaceSolid(belowPos, cube.FaceUp, tx)
			if !onGround {
				_, onGround = tx.Liquid(belowPos)
			}
		}

		if onGround {
			v.airTicks[id] = 0
		} else {
			v.airTicks[id]++
			// If ascending while long in air without flight permissions
			if v.airTicks[id] > v.conf.MaxAirTicks && dy > 0.05 {
				return false, fmt.Sprintf("fly violation: ascending in air without flight for %d ticks", v.airTicks[id])
			}
		}
	}

	return true, ""
}

// Reconcile performs authoritative rubber-banding by snapping the client back to validPos.
func (v *MovementValidator) Reconcile(s *Session, c Controllable, validPos mgl64.Vec3) {
	if s == nil || c == nil {
		return
	}
	s.ViewEntityTeleport(c, validPos)
}

// Clear cleans up tracked state for a disconnected player.
func (v *MovementValidator) Clear(id uuid.UUID) {
	if v == nil {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.airTicks != nil {
		delete(v.airTicks, id)
	}
}

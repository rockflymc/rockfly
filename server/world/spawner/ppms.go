package spawner

import (
	"math"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

// PlayerSnapshot represents an immutable value snapshot of a player's spatial state.
type PlayerSnapshot struct {
	ID  uuid.UUID
	Pos mgl64.Vec3
}

// TrackedMobSnapshot represents an entity evaluated during PPMS density calculations.
type TrackedMobSnapshot struct {
	ID       uuid.UUID
	Category MobCategory
	Pos      mgl64.Vec3
}

// PlayerMobBudget tracks the local mob count within a player's simulation sphere.
type PlayerMobBudget struct {
	PlayerID uuid.UUID
	Pos      mgl64.Vec3
	Hostile  int
	Passive  int
	Ambient  int
	Water    int
}

// Count returns the active count for the given category.
func (b *PlayerMobBudget) Count(cat MobCategory) int {
	switch cat {
	case CategoryHostile:
		return b.Hostile
	case CategoryPassive:
		return b.Passive
	case CategoryAmbient:
		return b.Ambient
	case CategoryWater:
		return b.Water
	default:
		return 0
	}
}

// Increment adds one to the category count.
func (b *PlayerMobBudget) Increment(cat MobCategory) {
	switch cat {
	case CategoryHostile:
		b.Hostile++
	case CategoryPassive:
		b.Passive++
	case CategoryAmbient:
		b.Ambient++
	case CategoryWater:
		b.Water++
	}
}

// PPMSConfig holds configuration parameters for Per-Player Mob Spawning.
type PPMSConfig struct {
	Enabled          bool
	Caps             MobCapSettings
	SimulationRadius float64 // default: 128 blocks
	MinSpawnRadius   float64 // default: 24 blocks
	MaxSpawnRadius   float64 // default: 54 blocks
	DespawnRadius    float64 // default: 128 blocks (instant despawn)
	SoftDespawnDist  float64 // default: 32 blocks
}

// DefaultPPMSConfig returns standard Paper-grade PPMS settings.
func DefaultPPMSConfig() PPMSConfig {
	return PPMSConfig{
		Enabled:          true,
		Caps:             DefaultMobCapSettings(),
		SimulationRadius: 128.0,
		MinSpawnRadius:   24.0,
		MaxSpawnRadius:   54.0,
		DespawnRadius:    128.0,
		SoftDespawnDist:  32.0,
	}
}

// PPMSEngine manages isolated per-player mob caps and density spheres,
// ensuring mob grinders in one area do not starve natural spawns across the rest of the world.
type PPMSEngine struct {
	mu   sync.RWMutex
	conf PPMSConfig
}

// NewPPMSEngine initializes a Per-Player Mob Spawner with the given configuration.
func NewPPMSEngine(conf PPMSConfig) *PPMSEngine {
	return &PPMSEngine{conf: conf}
}

// Config returns the active PPMS configuration.
func (e *PPMSEngine) Config() PPMSConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.conf
}

// EvaluateBudgets calculates isolated category mob budgets for each active player.
// Each entity is attributed to its nearest player within the simulation radius.
func (e *PPMSEngine) EvaluateBudgets(players []PlayerSnapshot, mobs []TrackedMobSnapshot) map[uuid.UUID]*PlayerMobBudget {
	e.mu.RLock()
	simRadiusSq := e.conf.SimulationRadius * e.conf.SimulationRadius
	e.mu.RUnlock()

	budgets := make(map[uuid.UUID]*PlayerMobBudget, len(players))
	for _, p := range players {
		budgets[p.ID] = &PlayerMobBudget{
			PlayerID: p.ID,
			Pos:      p.Pos,
		}
	}

	if len(players) == 0 {
		return budgets
	}

	for _, mob := range mobs {
		var nearestID uuid.UUID
		minDistSq := math.MaxFloat64

		for _, p := range players {
			dx := mob.Pos[0] - p.Pos[0]
			dy := mob.Pos[1] - p.Pos[1]
			dz := mob.Pos[2] - p.Pos[2]
			distSq := dx*dx + dy*dy + dz*dz

			if distSq < minDistSq && distSq <= simRadiusSq {
				minDistSq = distSq
				nearestID = p.ID
			}
		}

		if nearestID != uuid.Nil {
			if b, ok := budgets[nearestID]; ok {
				b.Increment(mob.Category)
			}
		}
	}

	return budgets
}

// CanSpawnForPlayer checks if the player's personal quota in category is not saturated.
func (e *PPMSEngine) CanSpawnForPlayer(playerID uuid.UUID, category MobCategory, budgets map[uuid.UUID]*PlayerMobBudget) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.conf.Enabled {
		return true
	}

	budget, ok := budgets[playerID]
	if !ok {
		return false
	}

	current := budget.Count(category)
	capLimit := e.capFor(category)

	return current < capLimit
}

func (e *PPMSEngine) capFor(cat MobCategory) int {
	switch cat {
	case CategoryHostile:
		return e.conf.Caps.Hostile
	case CategoryPassive:
		return e.conf.Caps.Passive
	case CategoryAmbient:
		return e.conf.Caps.Ambient
	case CategoryWater:
		return e.conf.Caps.Water
	default:
		return 0
	}
}

// ShouldDespawn evaluates despawn eligibility based on distance from nearest player.
// Returns (despawn bool, instant bool).
func (e *PPMSEngine) ShouldDespawn(mobPos mgl64.Vec3, players []PlayerSnapshot, ageTicks int64, rngFloat float64) (despawn bool, instant bool) {
	e.mu.RLock()
	despawnRadSq := e.conf.DespawnRadius * e.conf.DespawnRadius
	softDistSq := e.conf.SoftDespawnDist * e.conf.SoftDespawnDist
	e.mu.RUnlock()

	if len(players) == 0 {
		return true, true
	}

	closestDistSq := math.MaxFloat64
	for _, p := range players {
		dx := mobPos[0] - p.Pos[0]
		dy := mobPos[1] - p.Pos[1]
		dz := mobPos[2] - p.Pos[2]
		distSq := dx*dx + dy*dy + dz*dz
		if distSq < closestDistSq {
			closestDistSq = distSq
		}
	}

	// 1. Outside despawn radius (128 blocks) -> Instant despawn
	if closestDistSq > despawnRadSq {
		return true, true
	}

	// 2. Inside soft despawn distance (<= 32 blocks) -> Never despawn
	if closestDistSq <= softDistSq {
		return false, false
	}

	// 3. Between 32 and 128 blocks -> 1/800 chance per tick after 30 seconds (600 ticks)
	if ageTicks > 600 && rngFloat < (1.0/800.0) {
		return true, false
	}

	return false, false
}

// InPlayerSpawnRing verifies if a candidate coordinate is within [minSpawnRadius, maxSpawnRadius] of player.
func (e *PPMSEngine) InPlayerSpawnRing(candidate cube.Pos, playerPos mgl64.Vec3) bool {
	e.mu.RLock()
	minR := e.conf.MinSpawnRadius
	maxR := e.conf.MaxSpawnRadius
	e.mu.RUnlock()

	candVec := candidate.Vec3()
	dx := candVec[0] - playerPos[0]
	dz := candVec[2] - playerPos[2]
	dist := math.Sqrt(dx*dx + dz*dz)

	return dist >= minR && dist <= maxR
}

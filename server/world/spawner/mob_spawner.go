package spawner

import (
	"math"
	"math/rand"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// MobCategory classifies entities for population cap calculation.
type MobCategory int

const (
	// CategoryHostile represents aggressive monsters (zombies, skeletons, creepers, etc.).
	CategoryHostile MobCategory = iota
	// CategoryPassive represents peaceful livestock (cows, sheep, pigs, chickens, etc.).
	CategoryPassive
	// CategoryAmbient represents atmospheric creatures (bats, etc.).
	CategoryAmbient
	// CategoryWater represents aquatic creatures (fish, squid, dolphins, etc.).
	CategoryWater
)

// MobCapSettings defines the maximum entity counts per category per active player.
type MobCapSettings struct {
	Hostile int
	Passive int
	Ambient int
	Water   int
}

// DefaultMobCapSettings returns canonical Bedrock/Paper limits.
func DefaultMobCapSettings() MobCapSettings {
	return MobCapSettings{
		Hostile: 70,
		Passive: 10,
		Ambient: 15,
		Water:   5,
	}
}

// SpawnCandidate represents an evaluated coordinate suitable for mob placement.
type SpawnCandidate struct {
	Pos      cube.Pos
	Category MobCategory
	MobType  string
}

// MobSpawner orchestrates natural environmental mob populating according to Bedrock 1.18+
// lighting rules, distance spawning rings, and per-player mob caps.
type MobSpawner struct {
	caps           MobCapSettings
	minSpawnRadius float64
	maxSpawnRadius float64
}

// NewMobSpawner initializes a MobSpawner with default Bedrock caps and distance bounds.
func NewMobSpawner(caps MobCapSettings) *MobSpawner {
	return &MobSpawner{
		caps:           caps,
		minSpawnRadius: 24.0, // Minimum 24 blocks from any player
		maxSpawnRadius: 54.0, // Maximum 54 blocks (simulation distance 4 ring)
	}
}

// Caps returns the active category capacity configuration.
func (s *MobSpawner) Caps() MobCapSettings {
	return s.caps
}

// InSpawnRing returns true if candidatePos is within [minRadius, maxRadius] from playerPos.
func (s *MobSpawner) InSpawnRing(candidatePos, playerPos mgl64.Vec3) bool {
	dist := candidatePos.Sub(playerPos).Len()
	return dist >= s.minSpawnRadius && dist <= s.maxSpawnRadius
}

// CanSpawnHostile checks 1.18+ hostile rules: block light == 0, solid floor, clear headroom.
func (s *MobSpawner) CanSpawnHostile(tx *world.Tx, pos cube.Pos, playerDist float64) bool {
	if playerDist < s.minSpawnRadius || playerDist > s.maxSpawnRadius {
		return false
	}
	if tx == nil {
		return true
	}

	// 1. Light check: BlockLight must be 0 for monsters in Caves & Cliffs 1.18+
	if tx.BlockLight(pos) > 0 {
		return false
	}

	// 2. Headroom clearance: Target block and block above must not be solid
	bSpawn := tx.Block(pos)
	bAbove := tx.Block(pos.Add(cube.Pos{0, 1, 0}))
	if isSolid(tx, bSpawn, pos) || isSolid(tx, bAbove, pos.Add(cube.Pos{0, 1, 0})) {
		return false
	}

	// 3. Floor check: Supporting block underneath must be solid and non-liquid
	bFloor := tx.Block(pos.Side(cube.FaceDown))
	return isSolid(tx, bFloor, pos.Side(cube.FaceDown))
}

// CanSpawnPassive checks passive animal rules: daylight sky light >= 9, grass floor, clear headroom.
func (s *MobSpawner) CanSpawnPassive(tx *world.Tx, pos cube.Pos, playerDist float64) bool {
	if playerDist < s.minSpawnRadius || playerDist > s.maxSpawnRadius {
		return false
	}
	if tx == nil {
		return true
	}

	// 1. Sky light check: Animals require sunlight (SkyLight >= 9)
	if tx.SkyLight(pos) < 9 {
		return false
	}

	// 2. Floor must be grass block
	bFloor := tx.Block(pos.Side(cube.FaceDown))
	if _, ok := bFloor.(block.Grass); !ok {
		return false
	}

	// 3. Headroom clearance
	bSpawn := tx.Block(pos)
	bAbove := tx.Block(pos.Add(cube.Pos{0, 1, 0}))
	return !isSolid(tx, bSpawn, pos) && !isSolid(tx, bAbove, pos.Add(cube.Pos{0, 1, 0}))
}

// GenerateCandidateCoordinates samples randomized spawn positions around a center player.
func (s *MobSpawner) GenerateCandidateCoordinates(playerPos mgl64.Vec3, count int) []cube.Pos {
	candidates := make([]cube.Pos, 0, count)

	for i := 0; i < count; i++ {
		// Random distance in [24.0, 54.0]
		dist := s.minSpawnRadius + rand.Float64()*(s.maxSpawnRadius-s.minSpawnRadius)
		angle := rand.Float64() * 2 * math.Pi

		candidateX := playerPos[0] + dist*math.Cos(angle)
		candidateZ := playerPos[2] + dist*math.Sin(angle)
		candidateY := playerPos[1] + (rand.Float64()*16 - 8)

		candidates = append(candidates, cube.Pos{
			int(math.Floor(candidateX)),
			int(math.Floor(candidateY)),
			int(math.Floor(candidateZ)),
		})
	}
	return candidates
}

// CapExceeded checks whether the current population in category exceeds per-player limits.
func (s *MobSpawner) CapExceeded(category MobCategory, currentCount int, playerCount int) bool {
	if playerCount <= 0 {
		playerCount = 1
	}

	switch category {
	case CategoryHostile:
		return currentCount >= s.caps.Hostile*playerCount
	case CategoryPassive:
		return currentCount >= s.caps.Passive*playerCount
	case CategoryAmbient:
		return currentCount >= s.caps.Ambient*playerCount
	case CategoryWater:
		return currentCount >= s.caps.Water*playerCount
	default:
		return false
	}
}

// isSolid checks if a block has solid collision bounding boxes.
func isSolid(tx *world.Tx, b world.Block, pos cube.Pos) bool {
	if b == nil {
		return false
	}
	return len(b.Model().BBox(pos, tx)) > 0
}

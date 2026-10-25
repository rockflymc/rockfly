package spawner_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/spawner"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

func TestMobSpawner_DistanceRing(t *testing.T) {
	s := spawner.NewMobSpawner(spawner.DefaultMobCapSettings())
	playerPos := mgl64.Vec3{0, 64, 0}

	// Inside minimum radius (< 24 blocks)
	closePos := mgl64.Vec3{10, 64, 0}
	if s.InSpawnRing(closePos, playerPos) {
		t.Fatalf("expected closePos inside 24 blocks to be excluded from spawn ring")
	}

	// In valid ring [24.0, 54.0]
	validPos := mgl64.Vec3{32, 64, 0}
	if !s.InSpawnRing(validPos, playerPos) {
		t.Fatalf("expected validPos at 32 blocks to be within spawn ring")
	}

	// Outside maximum radius (> 54 blocks)
	farPos := mgl64.Vec3{60, 64, 0}
	if s.InSpawnRing(farPos, playerPos) {
		t.Fatalf("expected farPos at 60 blocks to be excluded from spawn ring")
	}
}

func TestMobSpawner_PerPlayerMobCaps(t *testing.T) {
	s := spawner.NewMobSpawner(spawner.DefaultMobCapSettings())

	// 1 player: cap is 70 hostiles
	if s.CapExceeded(spawner.CategoryHostile, 69, 1) {
		t.Fatalf("expected count 69 to not exceed cap 70 for 1 player")
	}
	if !s.CapExceeded(spawner.CategoryHostile, 70, 1) {
		t.Fatalf("expected count 70 to exceed cap 70 for 1 player")
	}

	// 3 players: cap scales to 210 hostiles
	if s.CapExceeded(spawner.CategoryHostile, 150, 3) {
		t.Fatalf("expected count 150 to not exceed cap 210 for 3 players")
	}
	if !s.CapExceeded(spawner.CategoryHostile, 210, 3) {
		t.Fatalf("expected count 210 to exceed cap 210 for 3 players")
	}

	// Passive animals: 10 per player
	if s.CapExceeded(spawner.CategoryPassive, 9, 1) {
		t.Fatalf("expected count 9 to not exceed passive cap 10 for 1 player")
	}
	if !s.CapExceeded(spawner.CategoryPassive, 10, 1) {
		t.Fatalf("expected count 10 to exceed passive cap 10 for 1 player")
	}
}

func TestMobSpawner_CandidateGeneration(t *testing.T) {
	s := spawner.NewMobSpawner(spawner.DefaultMobCapSettings())
	playerPos := mgl64.Vec3{100, 70, 100}

	candidates := s.GenerateCandidateCoordinates(playerPos, 50)
	if len(candidates) != 50 {
		t.Fatalf("expected 50 candidates, got %d", len(candidates))
	}

	for _, c := range candidates {
		v := mgl64.Vec3{float64(c.X()) + 0.5, float64(c.Y()) + 0.5, float64(c.Z()) + 0.5}
		// Horizontal distance to player
		hDist := mgl64.Vec2{v[0] - playerPos[0], v[2] - playerPos[2]}.Len()
		if hDist < 22.0 || hDist > 56.0 {
			t.Fatalf("candidate horizontal distance out of bounds: %f", hDist)
		}
	}
}

func TestPPMS_IsolatedPlayerBudgets(t *testing.T) {
	engine := spawner.NewPPMSEngine(spawner.DefaultPPMSConfig())

	pA := spawner.PlayerSnapshot{ID: uuid.New(), Pos: mgl64.Vec3{0, 64, 0}}
	pB := spawner.PlayerSnapshot{ID: uuid.New(), Pos: mgl64.Vec3{1000, 64, 1000}}
	players := []spawner.PlayerSnapshot{pA, pB}

	// Create 70 hostile mobs around Player A (a mob grinder trap)
	mobs := make([]spawner.TrackedMobSnapshot, 0, 70)
	for i := 0; i < 70; i++ {
		mobs = append(mobs, spawner.TrackedMobSnapshot{
			ID:       uuid.New(),
			Category: spawner.CategoryHostile,
			Pos:      mgl64.Vec3{float64(i%5 + 10), 64, float64(i%5 + 10)},
		})
	}

	budgets := engine.EvaluateBudgets(players, mobs)

	// Player A has 70 hostiles -> saturated, cannot spawn
	if engine.CanSpawnForPlayer(pA.ID, spawner.CategoryHostile, budgets) {
		t.Fatalf("expected Player A to be saturated at 70 hostiles")
	}

	// Player B has 0 hostiles -> open, CAN spawn!
	if !engine.CanSpawnForPlayer(pB.ID, spawner.CategoryHostile, budgets) {
		t.Fatalf("expected Player B with 0 nearby hostiles to be eligible for mob spawns")
	}
	if count := budgets[pB.ID].Hostile; count != 0 {
		t.Fatalf("expected Player B hostile count 0, got %d", count)
	}
}

func TestPPMS_DespawnRules(t *testing.T) {
	engine := spawner.NewPPMSEngine(spawner.DefaultPPMSConfig())
	player := spawner.PlayerSnapshot{ID: uuid.New(), Pos: mgl64.Vec3{0, 64, 0}}
	players := []spawner.PlayerSnapshot{player}

	// 1. Within 20 blocks: never despawn
	despawn, instant := engine.ShouldDespawn(mgl64.Vec3{15, 64, 0}, players, 1000, 0.0001)
	if despawn || instant {
		t.Fatalf("expected entity within 20 blocks to never despawn")
	}

	// 2. Beyond 128 blocks: instant hard despawn
	despawn, instant = engine.ShouldDespawn(mgl64.Vec3{135, 64, 0}, players, 100, 0.5)
	if !despawn || !instant {
		t.Fatalf("expected entity beyond 128 blocks to instantly hard despawn")
	}

	// 3. Between 32 and 128 blocks: soft despawn if age > 600 ticks and roll < 1/800
	despawn, instant = engine.ShouldDespawn(mgl64.Vec3{50, 64, 0}, players, 700, 0.0001)
	if !despawn || instant {
		t.Fatalf("expected entity at 50 blocks with roll < 1/800 to soft despawn")
	}

	// 4. Between 32 and 128 blocks: does NOT despawn if roll > 1/800
	despawn, _ = engine.ShouldDespawn(mgl64.Vec3{50, 64, 0}, players, 700, 0.5)
	if despawn {
		t.Fatalf("expected entity at 50 blocks with high roll to not despawn")
	}
}

func BenchmarkPPMS_EvaluateBudgets(b *testing.B) {
	engine := spawner.NewPPMSEngine(spawner.DefaultPPMSConfig())

	// 20 players
	players := make([]spawner.PlayerSnapshot, 20)
	for i := 0; i < 20; i++ {
		players[i] = spawner.PlayerSnapshot{
			ID:  uuid.New(),
			Pos: mgl64.Vec3{float64(i * 150), 64, float64(i * 150)},
		}
	}

	// 1,000 mobs
	mobs := make([]spawner.TrackedMobSnapshot, 1000)
	for i := 0; i < 1000; i++ {
		mobs[i] = spawner.TrackedMobSnapshot{
			ID:       uuid.New(),
			Category: spawner.MobCategory(i % 4),
			Pos:      mgl64.Vec3{float64((i * 23) % 3000), 64, float64((i * 37) % 3000)},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = engine.EvaluateBudgets(players, mobs)
	}
}

func BenchmarkMobSpawner_Evaluation(b *testing.B) {
	s := spawner.NewMobSpawner(spawner.DefaultMobCapSettings())
	pos := cube.Pos{10, 64, 10}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = s.CanSpawnHostile(nil, pos, 30.0)
		_ = s.CanSpawnPassive(nil, pos, 30.0)
	}
}

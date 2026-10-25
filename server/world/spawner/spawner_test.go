package spawner_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/spawner"
	"github.com/go-gl/mathgl/mgl64"
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
		v := mgl64.Vec3{float64(c.X()), float64(c.Y()), float64(c.Z())}
		// Horizontal distance to player
		hDist := mgl64.Vec2{v[0] - playerPos[0], v[2] - playerPos[2]}.Len()
		if hDist < 23.0 || hDist > 55.0 {
			t.Fatalf("candidate horizontal distance out of bounds: %f", hDist)
		}
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

package climate_test

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/generator/climate"
)

func TestParametersDistance(t *testing.T) {
	p1 := climate.Parameters{Temperature: 0.5, Humidity: 0.5}
	p2 := climate.Parameters{Temperature: 0.5, Humidity: 0.5}
	if p1.DistanceSq(p2) != 0.0 {
		t.Fatalf("expected 0 distance between identical points, got %v", p1.DistanceSq(p2))
	}

	p3 := climate.Parameters{Temperature: 1.5, Humidity: 0.5}
	if math.Abs(p1.DistanceSq(p3)-1.0) > 1e-6 {
		t.Fatalf("expected distance 1.0, got %v", p1.DistanceSq(p3))
	}
}

func TestClimateSamplerDeterminism(t *testing.T) {
	s1 := climate.NewSampler(12345)
	s2 := climate.NewSampler(12345)

	for x := -500; x <= 500; x += 250 {
		for z := -500; z <= 500; z += 250 {
			p1 := s1.Sample(x, 64, z)
			p2 := s2.Sample(x, 64, z)
			if p1 != p2 {
				t.Fatalf("expected identical climate parameters at (%d, %d): %v != %v", x, z, p1, p2)
			}

			b1 := s1.BiomeAt(x, 64, z)
			b2 := s2.BiomeAt(x, 64, z)
			if b1.EncodeBiome() != b2.EncodeBiome() {
				t.Fatalf("expected identical biome at (%d, %d): %d != %d", x, z, b1.EncodeBiome(), b2.EncodeBiome())
			}
		}
	}
}

func TestSubterraneanBiomes(t *testing.T) {
	s := climate.NewSampler(42)

	// High subterranean depth (Y = -50)
	surfaceParam := s.Sample(0, 70, 0)
	caveParam := s.Sample(0, -50, 0)

	if caveParam.Depth <= surfaceParam.Depth {
		t.Fatalf("expected cave depth (%v) to exceed surface depth (%v)", caveParam.Depth, surfaceParam.Depth)
	}
}

func TestVoronoiBiomeAt(t *testing.T) {
	s := climate.NewSampler(98765)
	v := climate.NewVoronoi(s, 98765)

	b1 := v.BiomeAt(100, 64, 100)
	if b1 == nil || b1.EncodeBiome() < 0 {
		t.Fatalf("invalid biome returned: %v", b1)
	}

	// Verify continuous evaluation across a short strip
	for x := 100; x <= 120; x++ {
		b := v.BiomeAt(x, 64, 100)
		if b == nil {
			t.Fatalf("nil biome returned at x=%d", x)
		}
	}
}

func TestEmptyTargetsFallback(t *testing.T) {
	s := climate.NewSamplerWithTargets(42, []climate.Target{})
	b := s.ClosestBiome(climate.Parameters{})
	if b == nil {
		t.Fatalf("expected non-nil default biome on empty targets")
	}
}

func TestOverworldTargetsCompleteness(t *testing.T) {
	targets := climate.OverworldTargets()
	if len(targets) < 20 {
		t.Fatalf("expected at least 20 climate targets, found %d", len(targets))
	}

	foundPlains := false
	foundDesert := false
	foundOcean := false
	foundRiver := false

	for _, target := range targets {
		switch target.Biome.(type) {
		case biome.Plains:
			foundPlains = true
		case biome.Desert:
			foundDesert = true
		case biome.Ocean:
			foundOcean = true
		case biome.River:
			foundRiver = true
		}
	}

	if !foundPlains || !foundDesert || !foundOcean || !foundRiver {
		t.Fatalf("missing essential biomes: plains=%v, desert=%v, ocean=%v, river=%v", foundPlains, foundDesert, foundOcean, foundRiver)
	}
}

func BenchmarkSamplerBiomeAt(b *testing.B) {
	s := climate.NewSampler(12345)
	b.ResetTimer()
	b.ReportAllocs()

	var sum int
	for i := range b.N {
		b := s.BiomeAt(i, 64, i)
		sum += b.EncodeBiome()
	}
	_ = sum
}

func BenchmarkVoronoiBiomeAt(b *testing.B) {
	s := climate.NewSampler(12345)
	v := climate.NewVoronoi(s, 12345)
	b.ResetTimer()
	b.ReportAllocs()

	var sum int
	for i := range b.N {
		b := v.BiomeAt(i, 64, i)
		sum += b.EncodeBiome()
	}
	_ = sum
}

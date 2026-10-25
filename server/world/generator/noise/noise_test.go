package noise_test

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/world/generator/noise"
)

func TestPerlinDeterminism(t *testing.T) {
	p1 := noise.NewPerlinFromSeed(12345678)
	p2 := noise.NewPerlinFromSeed(12345678)

	for x := -10.0; x <= 10.0; x += 1.5 {
		for z := -10.0; z <= 10.0; z += 1.5 {
			v1 := p1.Sample2D(x, z)
			v2 := p2.Sample2D(x, z)
			if v1 != v2 {
				t.Fatalf("expected deterministic output for (%.2f, %.2f): %v != %v", x, z, v1, v2)
			}
		}
	}
}

func TestPerlinContinuity(t *testing.T) {
	p := noise.NewPerlinFromSeed(42)

	// Step by very small increments and verify value delta is bounded
	prev := p.Sample3D(0, 0, 0)
	for x := 0.01; x <= 1.0; x += 0.01 {
		curr := p.Sample3D(x, x*0.5, x*0.2)
		delta := math.Abs(curr - prev)
		if delta > 0.25 {
			t.Fatalf("discontinuity detected at x=%.2f: delta=%.4f (prev=%.4f, curr=%.4f)", x, delta, prev, curr)
		}
		prev = curr
	}
}

func TestSimplex2DAnd3D(t *testing.T) {
	s1 := noise.NewSimplexFromSeed(9999)
	s2 := noise.NewSimplexFromSeed(9999)

	for x := -5.0; x <= 5.0; x += 1.0 {
		for y := -5.0; y <= 5.0; y += 1.0 {
			v1 := s1.Sample2D(x, y)
			v2 := s2.Sample2D(x, y)
			if v1 != v2 {
				t.Fatalf("expected identical 2D simplex samples at (%.1f, %.1f): %v != %v", x, y, v1, v2)
			}

			v3D_1 := s1.Sample3D(x, y, 0.5)
			v3D_2 := s2.Sample3D(x, y, 0.5)
			if v3D_1 != v3D_2 {
				t.Fatalf("expected identical 3D simplex samples at (%.1f, %.1f, 0.5): %v != %v", x, y, v3D_1, v3D_2)
			}
		}
	}
}

func TestOctaveNoise(t *testing.T) {
	oct := noise.NewOctavePerlinFromSeed(1337, 4, 2.0, 0.5)
	if oct.Octaves() != 4 {
		t.Fatalf("expected 4 octaves, got %d", oct.Octaves())
	}

	for x := -20.0; x <= 20.0; x += 4.0 {
		for z := -20.0; z <= 20.0; z += 4.0 {
			normalized := oct.SampleNormalized2D(x, z)
			if normalized < -1.0 || normalized > 1.0 {
				t.Fatalf("normalized noise out of range [-1, 1]: %v at (%.1f, %.1f)", normalized, x, z)
			}
		}
	}
}

func TestMojangOctaveNoise(t *testing.T) {
	r := noise.XoroshiroFromSeed(5555)
	oct := noise.NewMojangOctavePerlin(r, -4, 1.0, 1.0, 0.5, 0.25)
	if oct.Octaves() != 4 {
		t.Fatalf("expected 4 active octaves, got %d", oct.Octaves())
	}

	val := oct.Sample3D(100.0, 64.0, 200.0)
	if math.IsNaN(val) || math.IsInf(val, 0) {
		t.Fatalf("invalid noise sample: %v", val)
	}
}

func TestBatchSamplingParity(t *testing.T) {
	oct := noise.NewOctavePerlinFromSeed(777, 3, 2.0, 0.5)

	xs := []float64{0, 10, 20, 30}
	zs := []float64{5, 15, 25, 35}
	results := make([]float64, 4)

	oct.SampleBatch2D(xs, zs, results)

	for i := range 4 {
		expected := oct.Sample2D(xs[i], zs[i])
		if results[i] != expected {
			t.Fatalf("batch mismatch at index %d: batch=%v, single=%v", i, results[i], expected)
		}
	}

	var grid [16]float64
	oct.SampleBatch4x4(0, 0, 4.0, 4.0, &grid)

	var idx int
	for x := range 4 {
		rx := float64(x) * 4.0
		for z := range 4 {
			rz := float64(z) * 4.0
			expected := oct.Sample2D(rx, rz)
			if grid[idx] != expected {
				t.Fatalf("4x4 grid mismatch at [%d,%d]: grid=%v, single=%v", x, z, grid[idx], expected)
			}
			idx++
		}
	}
}

func TestBatch4WideParity(t *testing.T) {
	oct := noise.NewOctavePerlinFromSeed(999, 4, 2.0, 0.5)

	var xs, ys, zs, res2D, res3D [4]float64
	for i := range 4 {
		xs[i] = float64(i) * 12.5
		ys[i] = float64(i) * 8.2
		zs[i] = float64(i) * 15.7
	}

	oct.SampleBatch4Wide2D(&xs, &zs, &res2D)
	oct.SampleBatch4Wide3D(&xs, &ys, &zs, &res3D)

	for i := range 4 {
		exp2D := oct.Sample2D(xs[i], zs[i])
		if res2D[i] != exp2D {
			t.Fatalf("4-wide 2D mismatch at %d: got %v, expected %v", i, res2D[i], exp2D)
		}
		exp3D := oct.Sample3D(xs[i], ys[i], zs[i])
		if res3D[i] != exp3D {
			t.Fatalf("4-wide 3D mismatch at %d: got %v, expected %v", i, res3D[i], exp3D)
		}
	}
}

func BenchmarkOctaveSample2D(b *testing.B) {
	oct := noise.NewOctavePerlinFromSeed(42, 4, 2.0, 0.5)
	b.ResetTimer()
	b.ReportAllocs()

	var sum float64
	for i := range b.N {
		sum += oct.Sample2D(float64(i)*0.1, float64(i)*0.2)
	}
	_ = sum
}

func BenchmarkOctaveSampleBatch4x4(b *testing.B) {
	oct := noise.NewOctavePerlinFromSeed(42, 4, 2.0, 0.5)
	var grid [16]float64
	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		oct.SampleBatch4x4(float64(i), float64(i), 4.0, 4.0, &grid)
	}
}

func BenchmarkOctaveSampleCell4x4x4(b *testing.B) {
	oct := noise.NewOctavePerlinFromSeed(42, 4, 2.0, 0.5)
	var cell [64]float64
	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		oct.SampleCell4x4x4(float64(i), 64.0, float64(i), 4.0, 4.0, 4.0, &cell)
	}
}

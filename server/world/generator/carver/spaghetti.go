package carver

import (
	"github.com/df-mc/dragonfly/server/world/generator/noise"
)

// SpaghettiCarver generates winding, tubular subterranean tunnels ("spaghetti caves")
// formed by the intersection of two distinct 3D noise fields (n1^2 + n2^2 < r^2).
type SpaghettiCarver struct {
	noiseA *noise.OctaveNoise
	noiseB *noise.OctaveNoise
}

// NewSpaghettiCarver constructs a SpaghettiCarver for the given world seed.
func NewSpaghettiCarver(seed int64) *SpaghettiCarver {
	return &SpaghettiCarver{
		noiseA: noise.NewOctavePerlinFromSeed(seed^0x53706167, 3, 2.0, 0.5), // "Spag"
		noiseB: noise.NewOctavePerlinFromSeed(seed^0x68657474, 3, 2.0, 0.5), // "hett"
	}
}

// IsCave returns true if (x, y, z) falls within a spaghetti cave tunnel.
func (s *SpaghettiCarver) IsCave(x, y, z int) bool {
	if s == nil || s.noiseA == nil || s.noiseB == nil {
		return false
	}
	if y > 64 || y <= -60 {
		return false
	}

	fx := float64(x) * 0.02
	fy := float64(y) * 0.025
	fz := float64(z) * 0.02

	a := s.noiseA.SampleNormalized3D(fx, fy, fz)
	b := s.noiseB.SampleNormalized3D(fx, fy, fz)

	// Euclidean distance in 2D noise phase space forms continuous cylinder-like tunnels.
	distSq := a*a + b*b

	return distSq < 0.0049 // Radius approx 0.07 in normalized space (~2-4 blocks wide)
}

package carver

import (
	"github.com/df-mc/dragonfly/server/world/generator/noise"
)

// NoodleCarver generates narrow, twisting subterranean passages ("noodle caves")
// connecting larger cavern systems and surface ravines.
type NoodleCarver struct {
	noiseA *noise.OctaveNoise
	noiseB *noise.OctaveNoise
}

// NewNoodleCarver constructs a NoodleCarver for the given world seed.
func NewNoodleCarver(seed int64) *NoodleCarver {
	return &NoodleCarver{
		noiseA: noise.NewOctavePerlinFromSeed(seed^0x4e6f6f64, 4, 2.0, 0.5), // "Nood"
		noiseB: noise.NewOctavePerlinFromSeed(seed^0x6c657331, 4, 2.0, 0.5), // "les1"
	}
}

// IsCave returns true if (x, y, z) falls within a noodle cave tunnel.
func (n *NoodleCarver) IsCave(x, y, z int) bool {
	if y > 56 || y <= -58 {
		return false
	}

	// Higher frequency noise for tighter, more chaotic twists
	fx := float64(x) * 0.035
	fy := float64(y) * 0.035
	fz := float64(z) * 0.035

	a := n.noiseA.SampleNormalized3D(fx, fy, fz)
	b := n.noiseB.SampleNormalized3D(fx, fy, fz)

	distSq := a*a + b*b

	return distSq < 0.0016 // Radius approx 0.04 (~1-2 blocks wide)
}

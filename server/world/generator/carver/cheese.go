package carver

import (
	"github.com/df-mc/dragonfly/server/world/generator/noise"
)

// CheeseCarver evaluates large subterranean cavern chambers ("cheese caves")
// formed by thresholding 3D low-frequency Perlin noise.
type CheeseCarver struct {
	noise3D *noise.OctaveNoise
}

// NewCheeseCarver constructs a CheeseCarver for the given world seed.
func NewCheeseCarver(seed int64) *CheeseCarver {
	return &CheeseCarver{
		noise3D: noise.NewOctavePerlinFromSeed(seed^0x43686565, 3, 2.0, 0.5), // "Chee"
	}
}

// IsCave returns true if the subterranean coordinates (x, y, z) fall within a cheese cave chamber.
func (c *CheeseCarver) IsCave(x, y, z int) bool {
	// Cheese caves do not form in high atmosphere or below bedrock floor.
	if y > 60 || y <= -60 {
		return false
	}

	val := c.noise3D.SampleNormalized3D(float64(x)*0.015, float64(y)*0.02, float64(z)*0.015)

	// Threshold determines cavern fullness; values above 0.42 form wide subterranean hollows.
	return val > 0.42
}

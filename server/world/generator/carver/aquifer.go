package carver

import (
	"github.com/df-mc/dragonfly/server/world/generator/noise"
)

// AquiferSampler resolves the fluid (Air, Water, or Lava) occupying a carved subterranean voxel.
type AquiferSampler struct {
	waterLevelNoise *noise.OctaveNoise
	floodedNoise    *noise.OctaveNoise

	air   uint32
	water uint32
	lava  uint32
}

// NewAquiferSampler constructs an AquiferSampler with the given block runtime IDs and world seed.
func NewAquiferSampler(seed int64, air, water, lava uint32) *AquiferSampler {
	return &AquiferSampler{
		waterLevelNoise: noise.NewOctavePerlinFromSeed(seed^0x41717569, 2, 2.0, 0.5), // "Aqui"
		floodedNoise:    noise.NewOctavePerlinFromSeed(seed^0x466c6f6f, 3, 2.0, 0.5), // "Floo"
		air:             air,
		water:           water,
		lava:            lava,
	}
}

// FluidAt returns the block runtime ID occupying a carved void at (x, y, z).
func (a *AquiferSampler) FluidAt(x, y, z int) uint32 {
	if a == nil {
		return 0
	}
	// Deep cavern void below Y = -54 fills with subterranean lava sea
	if y <= -54 {
		return a.lava
	}

	// Flooding probability noise
	flooded := a.floodedNoise.SampleNormalized3D(float64(x)*0.01, float64(y)*0.01, float64(z)*0.01) > 0.25
	if !flooded {
		return a.air
	}

	// Local subterranean water table level
	levelSample := a.waterLevelNoise.SampleNormalized2D(float64(x)*0.008, float64(z)*0.008)
	localWaterLevel := int(30.0 + levelSample*30.0) // Ranges roughly from 0 to 60

	if y <= localWaterLevel {
		return a.water
	}

	return a.air
}

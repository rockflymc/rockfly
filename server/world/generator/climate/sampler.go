package climate

import (
	"math"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/generator/noise"
)

const (
	// ClimateHorizontalScale defines the spatial frequency scaling for 6D climate noise evaluation.
	ClimateHorizontalScale = 1.0 / 2048.0
	// ClimateVerticalScale defines the vertical scaling factor for subterranean biomes.
	ClimateVerticalScale = 1.0 / 128.0
)

// Sampler evaluates the 6-dimensional climate noise continuum from a world seed and maps
// arbitrary coordinates to authentic vanilla Overworld biomes.
type Sampler struct {
	tempNoise    *noise.OctaveNoise
	humidNoise   *noise.OctaveNoise
	contNoise    *noise.OctaveNoise
	erosionNoise *noise.OctaveNoise
	weirdNoise   *noise.OctaveNoise
	depthNoise   *noise.OctaveNoise
	targets      []Target
}

// NewSampler constructs a 6D climate sampler using the supplied world seed and canonical vanilla targets.
func NewSampler(seed int64) *Sampler {
	return NewSamplerWithTargets(seed, OverworldTargets())
}

// NewSamplerWithTargets constructs a 6D climate sampler using a custom set of climate targets.
func NewSamplerWithTargets(seed int64, targets []Target) *Sampler {
	return &Sampler{
		tempNoise:    noise.NewOctavePerlinFromSeed(seed^0x54656d70, 3, 2.0, 0.5), // "Temp"
		humidNoise:   noise.NewOctavePerlinFromSeed(seed^0x48756d69, 3, 2.0, 0.5), // "Humi"
		contNoise:    noise.NewOctavePerlinFromSeed(seed^0x436f6e74, 4, 2.0, 0.5), // "Cont"
		erosionNoise: noise.NewOctavePerlinFromSeed(seed^0x45726f73, 3, 2.0, 0.5), // "Eros"
		weirdNoise:   noise.NewOctavePerlinFromSeed(seed^0x57656972, 4, 2.0, 0.5), // "Weir"
		depthNoise:   noise.NewOctavePerlinFromSeed(seed^0x44657074, 2, 2.0, 0.5), // "Dept"
		targets:      targets,
	}
}

// Sample evaluates the 6 continuous climate parameters at the given world voxel coordinates.
func (s *Sampler) Sample(x, y, z int) Parameters {
	fx := float64(x) * ClimateHorizontalScale
	fz := float64(z) * ClimateHorizontalScale

	// Depth parameter scales with subterranean depth below sea level (Y=64).
	depthFactor := 0.0
	if y < 64 {
		depthFactor = float64(64-y) / 128.0
	}

	return Parameters{
		Temperature:     s.tempNoise.SampleNormalized2D(fx, fz),
		Humidity:        s.humidNoise.SampleNormalized2D(fx, fz),
		Continentalness: s.contNoise.SampleNormalized2D(fx, fz),
		Erosion:         s.erosionNoise.SampleNormalized2D(fx, fz),
		Weirdness:       s.weirdNoise.SampleNormalized2D(fx, fz),
		Depth:           depthFactor + s.depthNoise.SampleNormalized2D(fx, fz)*0.2,
	}
}

// BiomeAt returns the closest matching Biome in 6D climate space at the given world position.
func (s *Sampler) BiomeAt(x, y, z int) world.Biome {
	p := s.Sample(x, y, z)
	return s.ClosestBiome(p)
}

// ClosestBiome finds the Biome whose target parameters have the lowest squared Euclidean distance to p.
func (s *Sampler) ClosestBiome(p Parameters) world.Biome {
	var bestBiome world.Biome
	bestDist := math.MaxFloat64

	for i := range s.targets {
		t := &s.targets[i]
		d := p.DistanceSq(t.Params)
		if d < bestDist {
			bestDist = d
			bestBiome = t.Biome
		}
	}
	return bestBiome
}

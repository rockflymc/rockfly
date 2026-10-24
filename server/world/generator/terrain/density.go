package terrain

import (
	"github.com/df-mc/dragonfly/server/world/generator/climate"
	"github.com/df-mc/dragonfly/server/world/generator/noise"
)

// DensityEvaluator computes 3D terrain density values determining solid vs air voxels across the world.
type DensityEvaluator struct {
	contSpline     *Spline
	erosionSpline  *Spline
	climateSampler *climate.Sampler
	terrainNoise3D *noise.OctaveNoise
}

// NewDensityEvaluator constructs a 3D terrain density router for the given world seed and climate sampler.
func NewDensityEvaluator(seed int64, sampler *climate.Sampler) *DensityEvaluator {
	return &DensityEvaluator{
		contSpline:     CanonicalContinentalnessSpline(),
		erosionSpline:  CanonicalErosionSpline(),
		climateSampler: sampler,
		terrainNoise3D: noise.NewOctavePerlinFromSeed(seed^0x54657272, 4, 2.0, 0.5), // "Terr"
	}
}

// DensityAt computes the 3D density at voxel coordinates (x, y, z).
// Values >= 0 represent solid stone/terrain; values < 0 represent empty space / atmosphere / liquid.
func (d *DensityEvaluator) DensityAt(x, y, z int) float64 {
	params := d.climateSampler.Sample(x, y, z)

	baseHeight := d.contSpline.Compute(params.Continentalness) + d.erosionSpline.Compute(params.Erosion)

	// Sample 3D volumetric shaping noise for overhangs, cliffs, and floating terrain features.
	n3D := d.terrainNoise3D.Sample3D(float64(x)*0.015, float64(y)*0.015, float64(z)*0.015) * 16.0

	density := (baseHeight - float64(y)) + n3D

	// Apply vertical falloff dampening near the world height limits.
	if y > 256 {
		density -= float64(y-256) * 0.6
	} else if y < -54 {
		density += float64(-54-y) * 0.6
	}

	return density
}

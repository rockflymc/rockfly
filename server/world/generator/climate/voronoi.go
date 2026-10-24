package climate

import (
	"math"

	"github.com/df-mc/dragonfly/server/internal/mcrandom"
	"github.com/df-mc/dragonfly/server/world"
)

// Voronoi applies 3D cellular jitter to quarter-resolution biome grids to generate
// smooth, organic biome boundaries without 4x4 blocky staircase artifacts.
type Voronoi struct {
	sampler *Sampler
	seed    int64
}

// NewVoronoi constructs a Voronoi biome blender wrapping the supplied 6D climate sampler.
func NewVoronoi(sampler *Sampler, seed int64) *Voronoi {
	return &Voronoi{
		sampler: sampler,
		seed:    seed,
	}
}

// BiomeAt resolves the organic jittered Biome for the specified exact block voxel coordinates.
func (v *Voronoi) BiomeAt(x, y, z int) world.Biome {
	// Shift by 2 blocks to center cell boundaries on block coordinates.
	qx := (x - 2) >> 2
	qy := (y - 2) >> 2
	qz := (z - 2) >> 2

	fx := float64((x-2)&3) / 4.0
	fy := float64((y-2)&3) / 4.0
	fz := float64((z-2)&3) / 4.0

	var bestDX, bestDY, bestDZ int
	minDist := math.MaxFloat64

	// Inspect all 8 neighboring cell corners
	for dx := range 2 {
		for dy := range 2 {
			for dz := range 2 {
				jx, jy, jz := v.jitter(qx+dx, qy+dy, qz+dz)
				distX := fx - (float64(dx) + jx)
				distY := fy - (float64(dy) + jy)
				distZ := fz - (float64(dz) + jz)
				d := distX*distX + distY*distY + distZ*distZ

				if d < minDist {
					minDist = d
					bestDX = dx
					bestDY = dy
					bestDZ = dz
				}
			}
		}
	}

	// Sample the nearest cell center in quarter-resolution coordinates
	targetX := (qx + bestDX) << 2
	targetY := (qy + bestDY) << 2
	targetZ := (qz + bestDZ) << 2

	return v.sampler.BiomeAt(targetX, targetY, targetZ)
}

// jitter produces deterministic 3D displacement vector in [-0.45, 0.45] for cell (x, y, z).
func (v *Voronoi) jitter(x, y, z int) (float64, float64, float64) {
	h := mcrandom.MixStafford13(uint64(v.seed) ^ (uint64(x) * 0x1B873593) ^ 0x85EBCA6B ^ (uint64(y) * 0xCC9E2D51) ^ (uint64(z) * 0x52DCE729))

	jx := (float64((h)&1023)/1024.0 - 0.5) * 0.9
	jy := (float64((h>>10)&1023)/1024.0 - 0.5) * 0.9
	jz := (float64((h>>20)&1023)/1024.0 - 0.5) * 0.9

	return jx, jy, jz
}

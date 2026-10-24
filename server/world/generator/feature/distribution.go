package feature

import "math/rand/v2"

// Distribution computes random vertical Y heights for ore vein placement.
type Distribution interface {
	Sample(r *rand.Rand) int
}

// UniformDistribution places ores with equal probability across [minY, maxY].
type UniformDistribution struct {
	MinY int
	MaxY int
}

func (u UniformDistribution) Sample(r *rand.Rand) int {
	if u.MaxY <= u.MinY {
		return u.MinY
	}
	return u.MinY + r.IntN(u.MaxY-u.MinY+1)
}

// TriangleDistribution places ores with a triangular probability distribution
// peaking at (minY + maxY) / 2.
type TriangleDistribution struct {
	MinY int
	MaxY int
}

func (t TriangleDistribution) Sample(r *rand.Rand) int {
	if t.MaxY <= t.MinY {
		return t.MinY
	}
	half := (t.MaxY - t.MinY) / 2
	if half <= 0 {
		return t.MinY
	}
	return t.MinY + r.IntN(half+1) + r.IntN(half+1)
}

// InvertedTriangleDistribution places ores with maximum density at MinY,
// decreasing linearly towards MaxY.
type InvertedTriangleDistribution struct {
	MinY int
	MaxY int
}

func (it InvertedTriangleDistribution) Sample(r *rand.Rand) int {
	if it.MaxY <= it.MinY {
		return it.MinY
	}
	diff := it.MaxY - it.MinY
	// Minimum of two random integers creates a linear probability ramp towards 0
	val := r.IntN(diff + 1)
	val2 := r.IntN(diff + 1)
	if val2 < val {
		val = val2
	}
	return it.MinY + val
}

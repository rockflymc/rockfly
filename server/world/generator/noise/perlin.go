package noise

import (
	"math"

	"github.com/df-mc/dragonfly/server/internal/mcrandom"
)

// Random represents an abstract pseudo-random number generator for noise seeding.
type Random interface {
	Next() uint64
}

// XoroshiroFromSeed creates a Mojang-compatible Xoroshiro128PlusPlus PRNG from a 64-bit seed.
func XoroshiroFromSeed(seed int64) *mcrandom.Xoroshiro128PlusPlus {
	s0 := mcrandom.MixStafford13(uint64(seed))
	s1 := mcrandom.MixStafford13(uint64(seed) + 0x9E3779B97F4A7C15)
	return mcrandom.NewXoroshiro128PlusPlus(s0, s1)
}

// Perlin represents a Mojang-accurate single-octave 3D improved Perlin noise generator.
type Perlin struct {
	originX float64
	originY float64
	originZ float64
	perm    [512]uint8
}

// NewPerlin constructs a new Perlin noise generator using the supplied PRNG.
func NewPerlin(r Random) *Perlin {
	p := &Perlin{
		originX: float64(r.Next()&0xffffff) / 0xffffff * 256.0,
		originY: float64(r.Next()&0xffffff) / 0xffffff * 256.0,
		originZ: float64(r.Next()&0xffffff) / 0xffffff * 256.0,
	}

	var perm [256]uint8
	for i := range 256 {
		perm[i] = uint8(i)
	}

	for i := range 256 {
		j := int(r.Next() % uint64(256-i))
		perm[i], perm[i+j] = perm[i+j], perm[i]
	}

	for i := range 256 {
		p.perm[i] = perm[i]
		p.perm[i+256] = perm[i]
	}

	return p
}

// NewPerlinFromSeed creates a Perlin noise generator directly from an int64 seed.
func NewPerlinFromSeed(seed int64) *Perlin {
	return NewPerlin(XoroshiroFromSeed(seed))
}

// Sample2D samples 2D Perlin noise at (x, z).
func (p *Perlin) Sample2D(x, z float64) float64 {
	return p.Sample3D(x, 0, z)
}

// Sample3D samples 3D Perlin noise at (x, y, z).
func (p *Perlin) Sample3D(x, y, z float64) float64 {
	x += p.originX
	y += p.originY
	z += p.originZ

	xFloor := math.Floor(x)
	yFloor := math.Floor(y)
	zFloor := math.Floor(z)

	X := int(xFloor) & 255
	Y := int(yFloor) & 255
	Z := int(zFloor) & 255

	dx := x - xFloor
	dy := y - yFloor
	dz := z - zFloor

	u := fade(dx)
	v := fade(dy)
	w := fade(dz)

	A := int(p.perm[X]) + Y
	AA := int(p.perm[A]) + Z
	AB := int(p.perm[A+1]) + Z
	B := int(p.perm[X+1]) + Y
	BA := int(p.perm[B]) + Z
	BB := int(p.perm[B+1]) + Z

	return lerp(w,
		lerp(v,
			lerp(u, grad(p.perm[AA], dx, dy, dz), grad(p.perm[BA], dx-1, dy, dz)),
			lerp(u, grad(p.perm[AB], dx, dy-1, dz), grad(p.perm[BB], dx-1, dy-1, dz)),
		),
		lerp(v,
			lerp(u, grad(p.perm[AA+1], dx, dy, dz-1), grad(p.perm[BA+1], dx-1, dy, dz-1)),
			lerp(u, grad(p.perm[AB+1], dx, dy-1, dz-1), grad(p.perm[BB+1], dx-1, dy-1, dz-1)),
		),
	)
}

// fade computes the quintic polynomial curve 6t^5 - 15t^4 + 10t^3.
func fade(t float64) float64 {
	return t * t * t * (t*(t*6.0-15.0) + 10.0)
}

// lerp linearly interpolates between a and b by t.
func lerp(t, a, b float64) float64 {
	return a + t*(b-a)
}

// grad calculates gradient dot products branchlessly.
func grad(hash uint8, x, y, z float64) float64 {
	h := hash & 15
	u := x
	if h >= 8 {
		u = y
	}

	var v float64
	switch {
	case h < 4:
		v = y
	case h == 12 || h == 14:
		v = x
	default:
		v = z
	}

	res := u
	if h&1 != 0 {
		res = -u
	}
	if h&2 != 0 {
		res -= v
	} else {
		res += v
	}
	return res
}

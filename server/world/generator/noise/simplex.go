package noise

import (
	"math"
)

var (
	f2 = 0.5 * (math.Sqrt(3.0) - 1.0)
	g2 = (3.0 - math.Sqrt(3.0)) / 6.0
	f3 = 1.0 / 3.0
	g3 = 1.0 / 6.0
)

var simplexGradients3D = [12][3]float64{
	{1, 1, 0}, {-1, 1, 0}, {1, -1, 0}, {-1, -1, 0},
	{1, 0, 1}, {-1, 0, 1}, {1, 0, -1}, {-1, 0, -1},
	{0, 1, 1}, {0, -1, 1}, {0, 1, -1}, {0, -1, -1},
}

// Simplex provides high-speed 2D and 3D Simplex noise evaluation.
type Simplex struct {
	originX float64
	originY float64
	originZ float64
	perm    [512]uint8
}

// NewSimplex constructs a new Simplex noise generator using the supplied PRNG.
func NewSimplex(r Random) *Simplex {
	s := &Simplex{
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
		s.perm[i] = perm[i]
		s.perm[i+256] = perm[i]
	}

	return s
}

// NewSimplexFromSeed creates a Simplex noise generator from an int64 seed.
func NewSimplexFromSeed(seed int64) *Simplex {
	return NewSimplex(XoroshiroFromSeed(seed))
}

// Sample2D samples 2D Simplex noise at (x, y).
func (s *Simplex) Sample2D(x, y float64) float64 {
	x += s.originX
	y += s.originY

	skew := (x + y) * f2
	i := int(math.Floor(x + skew))
	j := int(math.Floor(y + skew))

	t := float64(i+j) * g2
	X0 := float64(i) - t
	Y0 := float64(j) - t
	x0 := x - X0
	y0 := y - Y0

	var i1, j1 int
	if x0 > y0 {
		i1, j1 = 1, 0
	} else {
		i1, j1 = 0, 1
	}

	x1 := x0 - float64(i1) + g2
	y1 := y0 - float64(j1) + g2
	x2 := x0 - 1.0 + 2.0*g2
	y2 := y0 - 1.0 + 2.0*g2

	ii := i & 255
	jj := j & 255

	gi0 := int(s.perm[ii+int(s.perm[jj])]) % 12
	gi1 := int(s.perm[ii+i1+int(s.perm[jj+j1])]) % 12
	gi2 := int(s.perm[ii+1+int(s.perm[jj+1])]) % 12

	var n0, n1, n2 float64

	t0 := 0.5 - x0*x0 - y0*y0
	if t0 > 0 {
		t0 *= t0
		n0 = t0 * t0 * (simplexGradients3D[gi0][0]*x0 + simplexGradients3D[gi0][1]*y0)
	}

	t1 := 0.5 - x1*x1 - y1*y1
	if t1 > 0 {
		t1 *= t1
		n1 = t1 * t1 * (simplexGradients3D[gi1][0]*x1 + simplexGradients3D[gi1][1]*y1)
	}

	t2 := 0.5 - x2*x2 - y2*y2
	if t2 > 0 {
		t2 *= t2
		n2 = t2 * t2 * (simplexGradients3D[gi2][0]*x2 + simplexGradients3D[gi2][1]*y2)
	}

	return 70.0 * (n0 + n1 + n2)
}

// Sample3D samples 3D Simplex noise at (x, y, z).
func (s *Simplex) Sample3D(x, y, z float64) float64 {
	x += s.originX
	y += s.originY
	z += s.originZ

	skew := (x + y + z) * f3
	i := int(math.Floor(x + skew))
	j := int(math.Floor(y + skew))
	k := int(math.Floor(z + skew))

	t := float64(i+j+k) * g3
	X0 := float64(i) - t
	Y0 := float64(j) - t
	Z0 := float64(k) - t
	x0 := x - X0
	y0 := y - Y0
	z0 := z - Z0

	var i1, j1, k1 int
	var i2, j2, k2 int

	if x0 >= y0 {
		if y0 >= z0 {
			i1, j1, k1 = 1, 0, 0
			i2, j2, k2 = 1, 1, 0
		} else if x0 >= z0 {
			i1, j1, k1 = 1, 0, 0
			i2, j2, k2 = 1, 0, 1
		} else {
			i1, j1, k1 = 0, 0, 1
			i2, j2, k2 = 1, 0, 1
		}
	} else {
		if y0 < z0 {
			i1, j1, k1 = 0, 0, 1
			i2, j2, k2 = 0, 1, 1
		} else if x0 < z0 {
			i1, j1, k1 = 0, 1, 0
			i2, j2, k2 = 0, 1, 1
		} else {
			i1, j1, k1 = 0, 1, 0
			i2, j2, k2 = 1, 1, 0
		}
	}

	x1 := x0 - float64(i1) + g3
	y1 := y0 - float64(j1) + g3
	z1 := z0 - float64(k1) + g3
	x2 := x0 - float64(i2) + 2.0*g3
	y2 := y0 - float64(j2) + 2.0*g3
	z2 := z0 - float64(k2) + 2.0*g3
	x3 := x0 - 1.0 + 3.0*g3
	y3 := y0 - 1.0 + 3.0*g3
	z3 := z0 - 1.0 + 3.0*g3

	ii := i & 255
	jj := j & 255
	kk := k & 255

	gi0 := int(s.perm[ii+int(s.perm[jj+int(s.perm[kk])])]) % 12
	gi1 := int(s.perm[ii+i1+int(s.perm[jj+j1+int(s.perm[kk+k1])])]) % 12
	gi2 := int(s.perm[ii+i2+int(s.perm[jj+j2+int(s.perm[kk+k2])])]) % 12
	gi3 := int(s.perm[ii+1+int(s.perm[jj+1+int(s.perm[kk+1])])]) % 12

	var n0, n1, n2, n3 float64

	t0 := 0.6 - x0*x0 - y0*y0 - z0*z0
	if t0 > 0 {
		t0 *= t0
		n0 = t0 * t0 * (simplexGradients3D[gi0][0]*x0 + simplexGradients3D[gi0][1]*y0 + simplexGradients3D[gi0][2]*z0)
	}

	t1 := 0.6 - x1*x1 - y1*y1 - z1*z1
	if t1 > 0 {
		t1 *= t1
		n1 = t1 * t1 * (simplexGradients3D[gi1][0]*x1 + simplexGradients3D[gi1][1]*y1 + simplexGradients3D[gi1][2]*z1)
	}

	t2 := 0.6 - x2*x2 - y2*y2 - z2*z2
	if t2 > 0 {
		t2 *= t2
		n2 = t2 * t2 * (simplexGradients3D[gi2][0]*x2 + simplexGradients3D[gi2][1]*y2 + simplexGradients3D[gi2][2]*z2)
	}

	t3 := 0.6 - x3*x3 - y3*y3 - z3*z3
	if t3 > 0 {
		t3 *= t3
		n3 = t3 * t3 * (simplexGradients3D[gi3][0]*x3 + simplexGradients3D[gi3][1]*y3 + simplexGradients3D[gi3][2]*z3)
	}

	return 32.0 * (n0 + n1 + n2 + n3)
}

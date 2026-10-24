package terrain

import (
	"cmp"
	"slices"
)

// SplinePoint represents a single knot in a Hermite cubic spline.
type SplinePoint struct {
	Location float64
	Value    float64
	Slope    float64
}

// Spline represents a smooth 1D Hermite cubic spline function.
type Spline struct {
	points []SplinePoint
}

// NewSpline constructs a new Spline from a set of knot points, sorting them by Location.
func NewSpline(points ...SplinePoint) *Spline {
	pts := make([]SplinePoint, len(points))
	copy(pts, points)
	slices.SortFunc(pts, func(a, b SplinePoint) int {
		return cmp.Compare(a.Location, b.Location)
	})
	return &Spline{points: pts}
}

// Compute evaluates the spline at x using cubic Hermite interpolation between knots
// and linear tangent extrapolation beyond the domain boundaries.
func (s *Spline) Compute(x float64) float64 {
	n := len(s.points)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return s.points[0].Value
	}

	first := s.points[0]
	if x <= first.Location {
		return first.Value + first.Slope*(x-first.Location)
	}

	last := s.points[n-1]
	if x >= last.Location {
		return last.Value + last.Slope*(x-last.Location)
	}

	// Find the bounding knot interval [i, i+1]
	var i int
	for idx := 0; idx < n-1; idx++ {
		if x < s.points[idx+1].Location {
			i = idx
			break
		}
	}

	p0 := s.points[i]
	p1 := s.points[i+1]

	delta := p1.Location - p0.Location
	if delta == 0 {
		return p0.Value
	}

	t := (x - p0.Location) / delta
	t2 := t * t
	t3 := t2 * t

	// Hermite basis polynomials
	h00 := 2.0*t3 - 3.0*t2 + 1.0
	h10 := t3 - 2.0*t2 + t
	h01 := -2.0*t3 + 3.0*t2
	h11 := t3 - t2

	return h00*p0.Value + h10*delta*p0.Slope + h01*p1.Value + h11*delta*p1.Slope
}

// CanonicalContinentalnessSpline maps Continentalness parameter to base terrain height.
func CanonicalContinentalnessSpline() *Spline {
	return NewSpline(
		SplinePoint{Location: -1.20, Value: 28.0, Slope: 25.0}, // Deep ocean trench
		SplinePoint{Location: -0.80, Value: 42.0, Slope: 30.0}, // Ocean floor
		SplinePoint{Location: -0.40, Value: 56.0, Slope: 25.0}, // Shallow ocean
		SplinePoint{Location: -0.19, Value: 62.0, Slope: 20.0}, // Shoreline / Beach
		SplinePoint{Location: 0.03, Value: 66.0, Slope: 20.0},  // Lowland plains
		SplinePoint{Location: 0.30, Value: 80.0, Slope: 40.0},  // Inland hills
		SplinePoint{Location: 0.60, Value: 110.0, Slope: 50.0}, // Highland plateaus
		SplinePoint{Location: 1.00, Value: 140.0, Slope: 60.0}, // Mountainous inland
	)
}

// CanonicalErosionSpline maps Erosion parameter to elevation modifier / jaggedness.
func CanonicalErosionSpline() *Spline {
	return NewSpline(
		SplinePoint{Location: -1.00, Value: 100.0, Slope: -120.0}, // Extreme mountain peaks
		SplinePoint{Location: -0.78, Value: 60.0, Slope: -80.0},   // Jagged slopes
		SplinePoint{Location: -0.375, Value: 25.0, Slope: -40.0},  // Mountain foothills
		SplinePoint{Location: 0.00, Value: 8.0, Slope: -15.0},     // Rolling hills
		SplinePoint{Location: 0.45, Value: 0.0, Slope: -5.0},      // Flat plains
		SplinePoint{Location: 1.00, Value: -8.0, Slope: 0.0},      // Low valleys
	)
}

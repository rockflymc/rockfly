package noise

// SampleBatch2D samples 2D octave noise across parallel slices into results.
// All slices must have the same length. Performs zero heap allocations.
func (o *OctaveNoise) SampleBatch2D(xs, zs, results []float64) {
	n := min(len(xs), len(zs), len(results))
	for i := range n {
		results[i] = o.Sample2D(xs[i], zs[i])
	}
}

// SampleBatch3D samples 3D octave noise across parallel slices into results.
// All slices must have the same length. Performs zero heap allocations.
func (o *OctaveNoise) SampleBatch3D(xs, ys, zs, results []float64) {
	n := min(len(xs), len(ys), len(zs), len(results))
	for i := range n {
		results[i] = o.Sample3D(xs[i], ys[i], zs[i])
	}
}

// SampleBatch4x4 samples a 4x4 grid of 2D noise points at quarter-resolution chunk coordinates.
// Results are populated into the provided 16-element array pointer with zero heap allocations.
func (o *OctaveNoise) SampleBatch4x4(startX, startZ, stepX, stepZ float64, results *[16]float64) {
	var idx int
	for x := range 4 {
		rx := startX + float64(x)*stepX
		for z := range 4 {
			rz := startZ + float64(z)*stepZ
			results[idx] = o.Sample2D(rx, rz)
			idx++
		}
	}
}

// SampleCell4x4x4 evaluates a 4x4x4 3D grid (64 sample points) for trilinear terrain density interpolation.
// Results are stored into the provided 64-element array pointer with zero heap allocations.
func (o *OctaveNoise) SampleCell4x4x4(startX, startY, startZ, stepX, stepY, stepZ float64, results *[64]float64) {
	var idx int
	for x := range 4 {
		rx := startX + float64(x)*stepX
		for y := range 4 {
			ry := startY + float64(y)*stepY
			for z := range 4 {
				rz := startZ + float64(z)*stepZ
				results[idx] = o.Sample3D(rx, ry, rz)
				idx++
			}
		}
	}
}

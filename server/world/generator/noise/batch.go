package noise

// SampleBatch4Wide2D computes 2D octave noise for 4 points simultaneously using unrolled evaluation.
// Minimizes loop overhead and maximizes register utilization with zero allocations.
func (o *OctaveNoise) SampleBatch4Wide2D(xs, zs *[4]float64, results *[4]float64) {
	var r0, r1, r2, r3 float64
	for i := range o.octaves {
		entry := &o.octaves[i]
		freq := entry.frequency
		amp := entry.amplitude
		r0 += entry.perlin.Sample2D(xs[0]*freq, zs[0]*freq) * amp
		r1 += entry.perlin.Sample2D(xs[1]*freq, zs[1]*freq) * amp
		r2 += entry.perlin.Sample2D(xs[2]*freq, zs[2]*freq) * amp
		r3 += entry.perlin.Sample2D(xs[3]*freq, zs[3]*freq) * amp
	}
	results[0] = r0
	results[1] = r1
	results[2] = r2
	results[3] = r3
}

// SampleBatch4Wide3D computes 3D octave noise for 4 points simultaneously using unrolled evaluation.
func (o *OctaveNoise) SampleBatch4Wide3D(xs, ys, zs *[4]float64, results *[4]float64) {
	var r0, r1, r2, r3 float64
	for i := range o.octaves {
		entry := &o.octaves[i]
		freq := entry.frequency
		amp := entry.amplitude
		r0 += entry.perlin.Sample3D(xs[0]*freq, ys[0]*freq, zs[0]*freq) * amp
		r1 += entry.perlin.Sample3D(xs[1]*freq, ys[1]*freq, zs[1]*freq) * amp
		r2 += entry.perlin.Sample3D(xs[2]*freq, ys[2]*freq, zs[2]*freq) * amp
		r3 += entry.perlin.Sample3D(xs[3]*freq, ys[3]*freq, zs[3]*freq) * amp
	}
	results[0] = r0
	results[1] = r1
	results[2] = r2
	results[3] = r3
}

// SampleBatch2D samples 2D octave noise across parallel slices into results.
// Employs 4-wide vector loop unrolling and performs zero heap allocations.
func (o *OctaveNoise) SampleBatch2D(xs, zs, results []float64) {
	n := min(len(xs), len(zs), len(results))
	i := 0
	var batchX, batchZ, batchRes [4]float64
	for ; i+4 <= n; i += 4 {
		batchX[0], batchX[1], batchX[2], batchX[3] = xs[i], xs[i+1], xs[i+2], xs[i+3]
		batchZ[0], batchZ[1], batchZ[2], batchZ[3] = zs[i], zs[i+1], zs[i+2], zs[i+3]
		o.SampleBatch4Wide2D(&batchX, &batchZ, &batchRes)
		results[i], results[i+1], results[i+2], results[i+3] = batchRes[0], batchRes[1], batchRes[2], batchRes[3]
	}
	for ; i < n; i++ {
		results[i] = o.Sample2D(xs[i], zs[i])
	}
}

// SampleBatch3D samples 3D octave noise across parallel slices into results.
// Employs 4-wide vector loop unrolling and performs zero heap allocations.
func (o *OctaveNoise) SampleBatch3D(xs, ys, zs, results []float64) {
	n := min(len(xs), len(ys), len(zs), len(results))
	i := 0
	var batchX, batchY, batchZ, batchRes [4]float64
	for ; i+4 <= n; i += 4 {
		batchX[0], batchX[1], batchX[2], batchX[3] = xs[i], xs[i+1], xs[i+2], xs[i+3]
		batchY[0], batchY[1], batchY[2], batchY[3] = ys[i], ys[i+1], ys[i+2], ys[i+3]
		batchZ[0], batchZ[1], batchZ[2], batchZ[3] = zs[i], zs[i+1], zs[i+2], zs[i+3]
		o.SampleBatch4Wide3D(&batchX, &batchY, &batchZ, &batchRes)
		results[i], results[i+1], results[i+2], results[i+3] = batchRes[0], batchRes[1], batchRes[2], batchRes[3]
	}
	for ; i < n; i++ {
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

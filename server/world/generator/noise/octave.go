package noise

import (
	"math"
)

// octaveEntry holds a single noise generator with its assigned frequency and amplitude.
type octaveEntry struct {
	perlin    *Perlin
	frequency float64
	amplitude float64
}

// OctaveNoise computes multi-frequency composite fractal noise over multiple octave layers.
type OctaveNoise struct {
	octaves        []octaveEntry
	totalAmplitude float64
}

// NewOctavePerlin creates a multi-octave Perlin noise generator using standard lacunarity and persistence.
// - octaves: number of octaves to generate (> 0).
// - lacunarity: frequency multiplier per octave (typically 2.0).
// - persistence: amplitude multiplier per octave (typically 0.5).
func NewOctavePerlin(r Random, octaves int, lacunarity, persistence float64) *OctaveNoise {
	if octaves <= 0 {
		octaves = 1
	}

	entries := make([]octaveEntry, octaves)
	var freq float64 = 1.0
	var amp float64 = 1.0
	var total float64

	for i := range octaves {
		entries[i] = octaveEntry{
			perlin:    NewPerlin(r),
			frequency: freq,
			amplitude: amp,
		}
		total += amp
		freq *= lacunarity
		amp *= persistence
	}

	return &OctaveNoise{
		octaves:        entries,
		totalAmplitude: total,
	}
}

// NewMojangOctavePerlin creates an OctaveNoise matching Mojang's 1.18+ climate noise parameterization.
// - firstOctave: the negative or zero power-of-two frequency exponent (e.g. -9 -> 2^-9).
// - amplitudes: amplitude weights for consecutive octaves. Zero weights are omitted.
func NewMojangOctavePerlin(r Random, firstOctave int, amplitudes ...float64) *OctaveNoise {
	var entries []octaveEntry
	var total float64

	for i, amp := range amplitudes {
		p := NewPerlin(r)
		if amp == 0 {
			continue
		}
		freq := math.Pow(2.0, float64(firstOctave+i))
		entries = append(entries, octaveEntry{
			perlin:    p,
			frequency: freq,
			amplitude: amp,
		})
		total += math.Abs(amp)
	}

	if len(entries) == 0 {
		entries = append(entries, octaveEntry{
			perlin:    NewPerlin(r),
			frequency: 1.0,
			amplitude: 1.0,
		})
		total = 1.0
	}

	return &OctaveNoise{
		octaves:        entries,
		totalAmplitude: total,
	}
}

// NewOctavePerlinFromSeed creates an OctaveNoise directly from an int64 world seed.
func NewOctavePerlinFromSeed(seed int64, octaves int, lacunarity, persistence float64) *OctaveNoise {
	return NewOctavePerlin(XoroshiroFromSeed(seed), octaves, lacunarity, persistence)
}

// Sample2D computes 2D multi-octave noise at (x, z).
func (o *OctaveNoise) Sample2D(x, z float64) float64 {
	var val float64
	for i := range o.octaves {
		entry := &o.octaves[i]
		val += entry.perlin.Sample2D(x*entry.frequency, z*entry.frequency) * entry.amplitude
	}
	return val
}

// Sample3D computes 3D multi-octave noise at (x, y, z).
func (o *OctaveNoise) Sample3D(x, y, z float64) float64 {
	var val float64
	for i := range o.octaves {
		entry := &o.octaves[i]
		val += entry.perlin.Sample3D(x*entry.frequency, y*entry.frequency, z*entry.frequency) * entry.amplitude
	}
	return val
}

// SampleNormalized2D samples 2D noise and divides by the total amplitude to keep output strictly within [-1.0, 1.0].
func (o *OctaveNoise) SampleNormalized2D(x, z float64) float64 {
	if o.totalAmplitude == 0 {
		return 0
	}
	return o.Sample2D(x, z) / o.totalAmplitude
}

// SampleNormalized3D samples 3D noise and divides by the total amplitude to keep output strictly within [-1.0, 1.0].
func (o *OctaveNoise) SampleNormalized3D(x, y, z float64) float64 {
	if o.totalAmplitude == 0 {
		return 0
	}
	return o.Sample3D(x, y, z) / o.totalAmplitude
}

// Octaves returns the number of active octaves.
func (o *OctaveNoise) Octaves() int {
	return len(o.octaves)
}

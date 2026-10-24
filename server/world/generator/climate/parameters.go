package climate

import (
	"github.com/df-mc/dragonfly/server/world"
)

// Parameters represents a point in Mojang's 6-dimensional climate space.
type Parameters struct {
	Temperature     float64
	Humidity        float64
	Continentalness float64
	Erosion         float64
	Weirdness       float64
	Depth           float64
}

// DistanceSq computes the squared Euclidean distance between two climate parameter points.
func (p Parameters) DistanceSq(other Parameters) float64 {
	dt := p.Temperature - other.Temperature
	dh := p.Humidity - other.Humidity
	dc := p.Continentalness - other.Continentalness
	de := p.Erosion - other.Erosion
	dw := p.Weirdness - other.Weirdness
	dd := p.Depth - other.Depth
	return dt*dt + dh*dh + dc*dc + de*de + dw*dw + dd*dd
}

// Target pairs a target climate parameter center point with its designated Biome.
type Target struct {
	Params Parameters
	Biome  world.Biome
}

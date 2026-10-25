package terrain

import (
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// CellGrid holds 3D density evaluation corner samples at 4x4x4 block intervals for a chunk column.
type CellGrid struct {
	corners [5][97][5]float64
}

// SampleGrid samples the 5x97x5 bounding corner points for a chunk at chunk coordinates (chunkX, chunkZ).
func (d *DensityEvaluator) SampleGrid(chunkX, chunkZ int, grid *CellGrid) {
	if d == nil || grid == nil {
		return
	}
	originX := chunkX << 4
	originZ := chunkZ << 4

	for cx := range 5 {
		wx := originX + (cx << 2)
		for cz := range 5 {
			wz := originZ + (cz << 2)
			for cy := range 97 {
				wy := -64 + (cy << 2)
				grid.corners[cx][cy][cz] = d.DensityAt(wx, wy, wz)
			}
		}
	}
}

// InterpolateChunk populates chunk blocks using trilinear interpolation across cell corners.
func InterpolateChunk(c *chunk.Chunk, grid *CellGrid, stone, deepslate, bedrock, water uint32) {
	if c == nil || grid == nil {
		return
	}
	for cx := range 4 {
		for cz := range 4 {
			for cy := range 96 {
				d000 := grid.corners[cx][cy][cz]
				d100 := grid.corners[cx+1][cy][cz]
				d010 := grid.corners[cx][cy+1][cz]
				d110 := grid.corners[cx+1][cy+1][cz]

				d001 := grid.corners[cx][cy][cz+1]
				d101 := grid.corners[cx+1][cy][cz+1]
				d011 := grid.corners[cx][cy+1][cz+1]
				d111 := grid.corners[cx+1][cy+1][cz+1]

				baseY := int16(-64 + (cy << 2))

				for dx := range 4 {
					tx := float64(dx) * 0.25
					x := uint8((cx << 2) + dx)

					// Lerp along X
					d00 := d000 + tx*(d100-d000)
					d10 := d010 + tx*(d110-d010)
					d01 := d001 + tx*(d101-d001)
					d11 := d011 + tx*(d111-d011)

					for dz := range 4 {
						tz := float64(dz) * 0.25
						z := uint8((cz << 2) + dz)

						// Lerp along Z
						d0 := d00 + tz*(d01-d00)
						d1 := d10 + tz*(d11-d10)

						for dy := range 4 {
							ty := float64(dy) * 0.25
							y := baseY + int16(dy)

							// Lerp along Y
							density := d0 + ty*(d1-d0)

							if density >= 0 {
								switch {
								case y == -64:
									c.SetBlock(x, y, z, 0, bedrock)
								case y < 0:
									c.SetBlock(x, y, z, 0, deepslate)
								default:
									c.SetBlock(x, y, z, 0, stone)
								}
							} else if y <= 63 {
								// Oceans and subterranean flooded areas below sea level
								c.SetBlock(x, y, z, 0, water)
							}
						}
					}
				}
			}
		}
	}
}

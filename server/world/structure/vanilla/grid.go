package vanilla

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// hashCoords computes a deterministic 64-bit integer hash from cell coordinates, seed, and salt.
func hashCoords(seed int64, cellX, cellZ int, salt int64) uint64 {
	h := uint64(seed) ^ (uint64(salt) * 0x9E3779B97F4A7C15)
	h ^= uint64(cellX) * 0xBF58476D1CE4E5B9
	h ^= uint64(cellZ) * 0x94D049BB133111EB
	h = (h ^ (h >> 30)) * 0xBF58476D1CE4E5B9
	h = (h ^ (h >> 27)) * 0x94D049BB133111EB
	return h ^ (h >> 31)
}

// floorDiv performs integer floor division consistent with negative coordinate spaces.
func floorDiv(a, b int) int {
	res := a / b
	rem := a % b
	if (a < 0) != (b < 0) && rem != 0 {
		res--
	}
	return res
}

// StructureChunkAtCell calculates the exact chunk coordinates of a structure within a grid cell.
func StructureChunkAtCell(seed int64, cellX, cellZ int, cfg PlacementConfig) (int, int) {
	if cfg.Spacing <= 0 {
		return cellX, cellZ
	}
	rangeLimit := cfg.Spacing - cfg.Separation
	if rangeLimit <= 0 {
		rangeLimit = 1
	}

	h := hashCoords(seed, cellX, cellZ, cfg.Salt)
	offsetX := int(h % uint64(rangeLimit))
	offsetZ := int((h >> 32) % uint64(rangeLimit))

	chunkX := cellX*cfg.Spacing + offsetX
	chunkZ := cellZ*cfg.Spacing + offsetZ
	return chunkX, chunkZ
}

// IsStructureChunk reports whether the specified chunk contains the start point of structure t.
func IsStructureChunk(seed int64, chunkPos world.ChunkPos, t Type) bool {
	cfg, ok := DefaultPlacements[t]
	if !ok {
		return false
	}
	if cfg.Spacing <= 1 {
		// Mineshafts have a 1% chance per chunk
		h := hashCoords(seed, int(chunkPos.X()), int(chunkPos.Z()), 0)
		return (h % 100) == 0
	}

	cellX := floorDiv(int(chunkPos.X()), cfg.Spacing)
	cellZ := floorDiv(int(chunkPos.Z()), cfg.Spacing)

	candX, candZ := StructureChunkAtCell(seed, cellX, cellZ, cfg)
	return int(chunkPos.X()) == candX && int(chunkPos.Z()) == candZ
}

// Locate searches in concentric square rings around origin for the nearest structure of type t.
// If isBiomeValid is non-nil, it checks whether candidate position resides in an acceptable biome.
// Returns the world block position and true if found, or (Pos{}, false) if not found within maxChunkRadius.
func Locate(seed int64, origin cube.Pos, t Type, maxChunkRadius int, isBiomeValid func(pos cube.Pos) bool) (cube.Pos, bool) {
	if maxChunkRadius <= 0 {
		maxChunkRadius = 64
	}
	if maxChunkRadius > 1024 {
		maxChunkRadius = 1024 // Cap search radius to prevent excessive search times
	}
	cfg, ok := DefaultPlacements[t]
	if !ok {
		return cube.Pos{}, false
	}

	originChunkX := origin.X() >> 4
	originChunkZ := origin.Z() >> 4

	cellSpacing := cfg.Spacing
	if cellSpacing <= 1 {
		cellSpacing = 16
	}

	maxCellRadius := (maxChunkRadius / cellSpacing) + 1
	originCellX := floorDiv(originChunkX, cellSpacing)
	originCellZ := floorDiv(originChunkZ, cellSpacing)

	var (
		bestPos  cube.Pos
		bestDist = math.MaxFloat64
		found    = false
	)

	// Search concentric rings of grid cells
	for r := 0; r <= maxCellRadius; r++ {
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				if intAbs(dx) != r && intAbs(dz) != r {
					continue // only inspect perimeter of the current ring
				}

				cellX := originCellX + dx
				cellZ := originCellZ + dz

				candChunkX, candChunkZ := StructureChunkAtCell(seed, cellX, cellZ, cfg)

				// Check chunk distance limit
				chunkDistX := candChunkX - originChunkX
				chunkDistZ := candChunkZ - originChunkZ
				if intAbs(chunkDistX) > maxChunkRadius || intAbs(chunkDistZ) > maxChunkRadius {
					continue
				}

				blockX := (candChunkX << 4) + 8
				blockZ := (candChunkZ << 4) + 8
				blockY := (cfg.MinY + cfg.MaxY) / 2
				candPos := cube.Pos{blockX, blockY, blockZ}

				if isBiomeValid != nil && !isBiomeValid(candPos) {
					continue
				}

				dx := candPos.X() - origin.X()
				dz := candPos.Z() - origin.Z()
				distSq := float64(dx*dx + dz*dz)
				if distSq < bestDist {
					bestDist = distSq
					bestPos = candPos
					found = true
				}
			}
		}

		// If a candidate was found in the current ring, return the closest candidate
		if found {
			return bestPos, true
		}
	}

	return cube.Pos{}, false
}

func intAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

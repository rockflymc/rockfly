package surface

// StratifyBedrock evaluates whether a coordinate (x, y, z) in the subterranean floor
// should be Bedrock based on authentic 1.18+ Bedrock Edition stratification dithering.
func StratifyBedrock(x, y, z int, seed int64) bool {
	// Y = -64 is the absolute floor of the world and must always be 100% solid bedrock.
	if y <= -64 {
		return true
	}
	// Bedrock floor only extends up to Y = -60.
	if y > -60 {
		return false
	}

	// Deterministic coordinate hash for smooth pseudo-random dithering without heap allocations.
	h := int64(x)*3129871 ^ int64(z)*116129781 ^ int64(y)*2654435761 ^ seed
	h = (h ^ (h >> 16)) * 0x45d9f3b
	h = (h ^ (h >> 16)) * 0x45d9f3b
	h = (h ^ (h >> 16)) & 0x7fffffff

	// Bedrock probability decreases linearly as Y ascends:
	// Y = -63: ~80%
	// Y = -62: ~60%
	// Y = -61: ~40%
	// Y = -60: ~20%
	threshold := int64(-60-y+1) * 200000000 // 200M, 400M, 600M, 800M out of 1B (approx max 0x7fffffff / 2.14)
	val := h % 1000000000

	return val < threshold
}

package surface

// DeepslateDither evaluates whether subterranean stone at Y in [0, 8] should be converted
// to Deepslate based on 1.18+ transition dithering.
func DeepslateDither(x, y, z int, seed int64) bool {
	if y < 0 {
		return true // 100% deepslate below Y=0
	}
	if y > 8 {
		return false // 100% stone above Y=8
	}

	// Deterministic coordinate hash for smooth dithering without allocations
	h := int64(x)*48271 ^ int64(z)*16807 ^ int64(y)*397204261 ^ seed
	h = (h ^ (h >> 13)) * 0x5bd1e995
	h = (h ^ (h >> 15)) & 0x7fffffff

	// Linear gradient: at Y=0 -> 8/8 (100%), at Y=4 -> 4/8 (50%), at Y=8 -> 0/8 (0%)
	threshold := int64(8-y) * 125000000 // 125M per step out of 1B
	val := h % 1000000000

	return val < threshold
}

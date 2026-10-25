//go:build !(wasip1 && wasm)

package rockfly

func rockfly_commit_block_batch(ptr uint32, count uint32) uint32 {
	return count
}

func rockfly_get_block(x int32, y int32, z int32) uint32 {
	return 0
}

func rockfly_log(level uint32, ptr uint32, length uint32) {
}

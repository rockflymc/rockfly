//go:build wasip1 && wasm

package rockfly

//go:wasmimport rockfly_env rockfly_commit_block_batch
func rockfly_commit_block_batch(ptr uint32, count uint32) uint32

//go:wasmimport rockfly_env rockfly_get_block
func rockfly_get_block(x int32, y int32, z int32) uint32

//go:wasmimport rockfly_env rockfly_log
func rockfly_log(level uint32, ptr uint32, length uint32)

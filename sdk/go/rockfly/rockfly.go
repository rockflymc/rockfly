package rockfly

import "unsafe"

// Log levels
const (
	LogLevelInfo  = 0
	LogLevelWarn  = 1
	LogLevelError = 2
)

// BlockMutation represents a single packed block placement (16 bytes).
type BlockMutation struct {
	X         int32
	Y         int32
	Z         int32
	RuntimeID uint32
}

// Log writes a message to the server console.
func Log(msg string) {
	LogWithLevel(LogLevelInfo, msg)
}

// LogWithLevel writes a message with a specific log level.
func LogWithLevel(level uint32, msg string) {
	if len(msg) == 0 {
		return
	}
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.StringData(msg))))
	rockfly_log(level, ptr, uint32(len(msg)))
}

// CommitBlockBatch commits an array of block mutations atomically to the server in a single world transaction.
// Returns the number of blocks successfully applied.
func CommitBlockBatch(mutations []BlockMutation) uint32 {
	if len(mutations) == 0 {
		return 0
	}
	ptr := uint32(uintptr(unsafe.Pointer(&mutations[0])))
	return rockfly_commit_block_batch(ptr, uint32(len(mutations)))
}

// GetBlock queries the runtime ID of the block at the specified coordinate.
func GetBlock(x, y, z int32) uint32 {
	return rockfly_get_block(x, y, z)
}

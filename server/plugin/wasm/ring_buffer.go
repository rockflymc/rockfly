package wasm

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// BlockMutation represents a single block placement or change inside a batch.
// Total packed size: 16 bytes (3x int32 + 1x uint32).
type BlockMutation struct {
	X         int32
	Y         int32
	Z         int32
	RuntimeID uint32
}

const (
	// BlockMutationSize is the byte size of each BlockMutation in linear memory.
	BlockMutationSize = 16

	// MaxBatchSize is the safety limit on the number of block mutations per commit.
	MaxBatchSize = 1_000_000
)

var (
	ErrOutOfBounds   = errors.New("wasm: memory access out of bounds")
	ErrBatchTooLarge = errors.New("wasm: batch size exceeds maximum limit")
)

// DecodeBlockBatch parses an array of BlockMutation structs directly from WASM linear memory
// with zero heap allocations for bounds checking.
func DecodeBlockBatch(mem []byte, ptr uint32, count uint32) ([]BlockMutation, error) {
	if count == 0 {
		return nil, nil
	}
	if count > MaxBatchSize {
		return nil, fmt.Errorf("%w: %d > %d", ErrBatchTooLarge, count, MaxBatchSize)
	}

	byteLen := uint64(count) * BlockMutationSize
	end := uint64(ptr) + byteLen

	if end > uint64(len(mem)) {
		return nil, fmt.Errorf("%w: range [%d, %d) exceeds memory size %d", ErrOutOfBounds, ptr, end, len(mem))
	}

	mutations := make([]BlockMutation, count)
	offset := int(ptr)

	for i := uint32(0); i < count; i++ {
		mutations[i] = BlockMutation{
			X:         int32(binary.LittleEndian.Uint32(mem[offset : offset+4])),
			Y:         int32(binary.LittleEndian.Uint32(mem[offset+4 : offset+8])),
			Z:         int32(binary.LittleEndian.Uint32(mem[offset+8 : offset+12])),
			RuntimeID: binary.LittleEndian.Uint32(mem[offset+12 : offset+16]),
		}
		offset += BlockMutationSize
	}

	return mutations, nil
}

// EncodeBlockBatch writes a slice of BlockMutations into a target byte slice at the given offset.
func EncodeBlockBatch(target []byte, offset int, mutations []BlockMutation) error {
	required := offset + len(mutations)*BlockMutationSize
	if required > len(target) {
		return fmt.Errorf("target buffer too small: required %d, capacity %d", required, len(target))
	}

	for _, m := range mutations {
		binary.LittleEndian.PutUint32(target[offset:offset+4], uint32(m.X))
		binary.LittleEndian.PutUint32(target[offset+4:offset+8], uint32(m.Y))
		binary.LittleEndian.PutUint32(target[offset+8:offset+12], uint32(m.Z))
		binary.LittleEndian.PutUint32(target[offset+12:offset+16], m.RuntimeID)
		offset += BlockMutationSize
	}
	return nil
}

// LinearRingBuffer represents a circular buffer in linear memory for streaming operations.
type LinearRingBuffer struct {
	Capacity uint32
	Head     uint32
	Tail     uint32
}

// NewLinearRingBuffer initializes a new ring buffer description with given capacity.
func NewLinearRingBuffer(capacity uint32) *LinearRingBuffer {
	return &LinearRingBuffer{
		Capacity: capacity,
	}
}

// Used returns the number of active items currently stored in the ring buffer.
func (r *LinearRingBuffer) Used() uint32 {
	if r.Head >= r.Tail {
		return r.Head - r.Tail
	}
	return r.Capacity - (r.Tail - r.Head)
}

// Available returns the number of free slots remaining in the ring buffer.
func (r *LinearRingBuffer) Available() uint32 {
	if r.Capacity == 0 {
		return 0
	}
	return r.Capacity - 1 - r.Used()
}

// Push reserves space for count items and returns the starting index.
func (r *LinearRingBuffer) Push(count uint32) (uint32, error) {
	if count > r.Available() {
		return 0, fmt.Errorf("%w: capacity %d, needed %d", ErrBatchTooLarge, r.Capacity, count)
	}
	start := r.Head
	r.Head = (r.Head + count) % r.Capacity
	return start, nil
}

// Advance commits/consumes count items from the buffer.
func (r *LinearRingBuffer) Advance(count uint32) error {
	if count > r.Used() {
		return fmt.Errorf("wasm ring buffer underflow: used %d, requested advance %d", r.Used(), count)
	}
	r.Tail = (r.Tail + count) % r.Capacity
	return nil
}

// Reset clears the ring buffer indices.
func (r *LinearRingBuffer) Reset() {
	r.Head = 0
	r.Tail = 0
}

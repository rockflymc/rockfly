package chunk

import (
	"sync"
)

// StandardPaletteCapacity is the default capacity for sub-chunk palette indices (4096 blocks).
const StandardPaletteCapacity = 4096

// PalettePool provides memory arenas for palette indices and temporary runtime ID slices,
// eliminating repeated slice allocations during sub-chunk encoding and decoding.
type PalettePool struct {
	indicesPool sync.Pool
	bytePool    sync.Pool
}

// GlobalPalettePool is the package-level shared palette memory pool.
var GlobalPalettePool = NewPalettePool()

// NewPalettePool creates an initialized palette memory pool.
func NewPalettePool() *PalettePool {
	return &PalettePool{
		indicesPool: sync.Pool{
			New: func() any {
				s := make([]uint32, StandardPaletteCapacity)
				return &s
			},
		},
		bytePool: sync.Pool{
			New: func() any {
				b := make([]byte, StandardPaletteCapacity*4)
				return &b
			},
		},
	}
}

// GetIndices acquires a []uint32 slice of length size (capped to capacity) from the pool.
func (p *PalettePool) GetIndices(size int) []uint32 {
	if size <= StandardPaletteCapacity {
		ptr := p.indicesPool.Get().(*[]uint32)
		slice := *ptr
		clear(slice[:size])
		return slice[:size]
	}
	return make([]uint32, size)
}

// PutIndices returns a borrowed []uint32 slice to the pool.
func (p *PalettePool) PutIndices(indices []uint32) {
	if cap(indices) >= StandardPaletteCapacity {
		indices = indices[:StandardPaletteCapacity]
		p.indicesPool.Put(&indices)
	}
}

// GetBytes acquires a []byte slice from the palette pool.
func (p *PalettePool) GetBytes(size int) []byte {
	if size <= StandardPaletteCapacity*4 {
		ptr := p.bytePool.Get().(*[]byte)
		slice := *ptr
		clear(slice[:size])
		return slice[:size]
	}
	return make([]byte, size)
}

// PutBytes returns a borrowed []byte slice to the palette pool.
func (p *PalettePool) PutBytes(b []byte) {
	if cap(b) >= StandardPaletteCapacity*4 {
		b = b[:StandardPaletteCapacity*4]
		p.bytePool.Put(&b)
	}
}

// AcquirePaletteIndices gets a palette slice from the global pool.
func AcquirePaletteIndices(size int) []uint32 {
	return GlobalPalettePool.GetIndices(size)
}

// ReleasePaletteIndices returns a palette slice to the global pool.
func ReleasePaletteIndices(b []uint32) {
	GlobalPalettePool.PutIndices(b)
}

package chunk

import (
	"sync"
)

// StandardPaletteCapacity is the default capacity for sub-chunk palette indices (4096 blocks).
const StandardPaletteCapacity = 4096

type paletteIndices = [StandardPaletteCapacity]uint32
type paletteBytes = [StandardPaletteCapacity * 4]byte

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
				return new(paletteIndices)
			},
		},
		bytePool: sync.Pool{
			New: func() any {
				return new(paletteBytes)
			},
		},
	}
}

// GetIndices acquires a []uint32 slice of length size (capped to capacity) from the pool.
func (p *PalettePool) GetIndices(size int) []uint32 {
	if p == nil {
		if size <= 0 {
			return nil
		}
		return make([]uint32, size)
	}
	if size <= 0 {
		return nil
	}
	if size <= StandardPaletteCapacity {
		ptr, ok := p.indicesPool.Get().(*paletteIndices)
		if !ok || ptr == nil {
			ptr = new(paletteIndices)
		}
		slice := ptr[:size]
		clear(slice)
		return slice
	}
	return make([]uint32, size)
}

// PutIndices returns a borrowed []uint32 slice to the pool.
func (p *PalettePool) PutIndices(indices []uint32) {
	if p == nil {
		return
	}
	if cap(indices) >= StandardPaletteCapacity {
		ptr := (*paletteIndices)(indices[:StandardPaletteCapacity])
		p.indicesPool.Put(ptr)
	}
}

// GetBytes acquires a []byte slice from the palette pool.
func (p *PalettePool) GetBytes(size int) []byte {
	if p == nil {
		if size <= 0 {
			return nil
		}
		return make([]byte, size)
	}
	if size <= 0 {
		return nil
	}
	if size <= StandardPaletteCapacity*4 {
		ptr, ok := p.bytePool.Get().(*paletteBytes)
		if !ok || ptr == nil {
			ptr = new(paletteBytes)
		}
		slice := ptr[:size]
		clear(slice)
		return slice
	}
	return make([]byte, size)
}

// PutBytes returns a borrowed []byte slice to the palette pool.
func (p *PalettePool) PutBytes(b []byte) {
	if p == nil {
		return
	}
	if cap(b) >= StandardPaletteCapacity*4 {
		ptr := (*paletteBytes)(b[:StandardPaletteCapacity*4])
		p.bytePool.Put(ptr)
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

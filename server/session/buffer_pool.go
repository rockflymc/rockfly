package session

import (
	"bytes"
	"sync"
)

// SmallBufferSize is the size of small packet buffers (1 KB).
const SmallBufferSize = 1024

// MediumBufferSize is the size of medium packet buffers (64 KB).
const MediumBufferSize = 64 * 1024

// LargeBufferSize is the size of large packet / chunk buffers (512 KB).
const LargeBufferSize = 512 * 1024

// BufferPool provides tiered sync.Pool memory arenas for byte buffers, eliminating
// heap allocation churn and GC pauses during high-frequency packet serialization and broadcasting.
type BufferPool struct {
	small   sync.Pool
	medium  sync.Pool
	large   sync.Pool
	bufPool sync.Pool
}

// NewBufferPool creates an initialized tiered buffer pool.
func NewBufferPool() *BufferPool {
	return &BufferPool{
		small: sync.Pool{
			New: func() any {
				b := make([]byte, SmallBufferSize)
				return &b
			},
		},
		medium: sync.Pool{
			New: func() any {
				b := make([]byte, MediumBufferSize)
				return &b
			},
		},
		large: sync.Pool{
			New: func() any {
				b := make([]byte, LargeBufferSize)
				return &b
			},
		},
		bufPool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, SmallBufferSize))
			},
		},
	}
}

// GlobalBufferPool is the package-level shared buffer pool.
var GlobalBufferPool = NewBufferPool()

// Get returns a byte slice of at least minCapacity from the appropriate tier pool.
func (p *BufferPool) Get(minCapacity int) []byte {
	switch {
	case minCapacity <= SmallBufferSize:
		ptr := p.small.Get().(*[]byte)
		return (*ptr)[:minCapacity]
	case minCapacity <= MediumBufferSize:
		ptr := p.medium.Get().(*[]byte)
		return (*ptr)[:minCapacity]
	case minCapacity <= LargeBufferSize:
		ptr := p.large.Get().(*[]byte)
		return (*ptr)[:minCapacity]
	default:
		return make([]byte, minCapacity)
	}
}

// Put returns a borrowed byte slice back to the appropriate tier pool.
func (p *BufferPool) Put(buf []byte) {
	c := cap(buf)
	switch {
	case c >= LargeBufferSize:
		buf = buf[:LargeBufferSize]
		p.large.Put(&buf)
	case c >= MediumBufferSize:
		buf = buf[:MediumBufferSize]
		p.medium.Put(&buf)
	case c >= SmallBufferSize:
		buf = buf[:SmallBufferSize]
		p.small.Put(&buf)
	}
}

// GetBuffer acquires a clean, empty *bytes.Buffer from the pool.
func (p *BufferPool) GetBuffer() *bytes.Buffer {
	buf := p.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer resets and returns a *bytes.Buffer to the pool.
func (p *BufferPool) PutBuffer(b *bytes.Buffer) {
	if b == nil {
		return
	}
	// Avoid retaining oversized buffers in the pool
	if b.Cap() > LargeBufferSize*2 {
		return
	}
	b.Reset()
	p.bufPool.Put(b)
}

// AcquireBuffer gets a buffer from the default global pool.
func AcquireBuffer() *bytes.Buffer {
	return GlobalBufferPool.GetBuffer()
}

// ReleaseBuffer returns a buffer to the default global pool.
func ReleaseBuffer(b *bytes.Buffer) {
	GlobalBufferPool.PutBuffer(b)
}

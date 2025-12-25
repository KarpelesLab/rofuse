package rofuse

import (
	"sync"

	"github.com/KarpelesLab/rofuse/proto"
)

// bufferPool manages a pool of reusable buffers for FUSE I/O.
type bufferPool struct {
	pool sync.Pool
	size int
}

// newBufferPool creates a new buffer pool with the specified buffer size.
func newBufferPool(size int) *bufferPool {
	if size < proto.MinBufferSize {
		size = proto.MinBufferSize
	}
	return &bufferPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, size)
				return &buf
			},
		},
	}
}

// get retrieves a buffer from the pool.
func (p *bufferPool) get() []byte {
	return *p.pool.Get().(*[]byte)
}

// put returns a buffer to the pool.
func (p *bufferPool) put(buf []byte) {
	// Only return buffers of the correct size
	if cap(buf) == p.size {
		buf = buf[:p.size]
		p.pool.Put(&buf)
	}
}

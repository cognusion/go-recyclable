package recyclable

import "sync"

// BufferPool is a self-managing pool of Buffers
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool returns an initialized BufferPool
func NewBufferPool() *BufferPool {
	var b BufferPool
	b.pool = sync.Pool{
		New: func() any {
			return NewBuffer(&b, make([]byte, 0))
		},
	}
	return &b
}

// Get will return an existing Buffer or a new one if the pool is empty.
// REMEMBER to Reset or Empty the Buffer and don't just start using it, as it
// may very well have old data in it!
func (p *BufferPool) Get() *Buffer {
	return p.pool.Get().(*Buffer)
}

// Put returns a Buffer to the pool. Generally calling Buffer.Close() is
// preferable to calling this directly.
func (p *BufferPool) Put(b *Buffer) {
	p.pool.Put(b)
}

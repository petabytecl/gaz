package tintx

import "sync"

// buffer is a byte slice that can be pooled for efficient allocation.
type buffer []byte

// bufPool is a sync.Pool for buffer reuse.
var bufPool = sync.Pool{
	New: func() any {
		b := make(buffer, 0, 1024)
		return &b
	},
}

// newBuffer retrieves a buffer from the pool.
func newBuffer() *buffer {
	return bufPool.Get().(*buffer)
}

// Free returns the buffer to the pool after resetting its length.
func (b *buffer) Free() {
	// Reset length, keep capacity
	*b = (*b)[:0]
	bufPool.Put(b)
}

// Write appends bytes to the buffer.
func (b *buffer) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

// WriteString appends a string to the buffer.
func (b *buffer) WriteString(s string) (int, error) {
	*b = append(*b, s...)
	return len(s), nil
}

// WriteByte appends a single byte to the buffer.
func (b *buffer) WriteByte(c byte) error {
	*b = append(*b, c)
	return nil
}

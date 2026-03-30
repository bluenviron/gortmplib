package bytecounter

import (
	"io"
	"sync/atomic"
)

// Reader allows to count read bytes.
type Reader struct {
	r     io.Reader
	count atomic.Uint64
}

// NewReader allocates a Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: r,
	}
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.count.Add(uint64(n))
	return n, err
}

// Count returns received bytes.
func (r *Reader) Count() uint64 {
	return r.count.Load()
}

// SetCount sets read bytes.
func (r *Reader) SetCount(v uint64) {
	r.count.Store(v)
}

// Package bufpool contains a pool of byte buffers.
package bufpool

import "sync"

// Writing a media frame needs a few buffers as big as the frame itself.
// Allocating them each time makes the garbage collector one of the main
// costs of writing media, especially with many destinations,
// so they are recycled.
var pool sync.Pool

// Get returns a buffer of length n. Its content is not zeroed:
// the caller must overwrite all of it.
func Get(n int) []byte {
	if p, ok := pool.Get().(*[]byte); ok && cap(*p) >= n {
		return (*p)[:n]
	}
	return make([]byte, n)
}

// Put gives a buffer back, to be reused by Get.
// The buffer must not be used afterwards.
func Put(b []byte) {
	if cap(b) != 0 {
		pool.Put(&b)
	}
}

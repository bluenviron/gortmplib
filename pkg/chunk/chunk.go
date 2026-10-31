// Package chunk implements RTMP chunks.
package chunk

import (
	"io"
)

// Chunk is a chunk.
type Chunk interface {
	Read(r io.Reader, bodyLen uint32, hasExtendedTimestamp bool) error
	MarshalHeaderTo(buf []byte, hasExtendedTimestamp bool) int
	GetBody() []byte

	// Deprecated: not used anymore.
	Marshal(hasExtendedTimestamp bool) ([]byte, error)
}

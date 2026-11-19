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

func readBody(r io.Reader, size uint32) ([]byte, error) {
	if size == 0 {
		return []byte{}, nil
	}
	body, err := io.ReadAll(io.LimitReader(r, int64(size)))
	if uint32(len(body)) == size {
		return body, nil
	}
	if err != nil {
		return body, err
	}
	if len(body) == 0 {
		return body, io.EOF
	}
	return body, io.ErrUnexpectedEOF
}

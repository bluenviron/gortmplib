package chunk

import (
	"io"
)

// Chunk3 is a type 3 chunk.
// Type 3 chunks have no message header. The stream ID, message length
// and timestamp delta fields are not present; chunks of this type take
// values from the preceding chunk for the same Chunk Stream ID. When a
// single message is split into chunks, all chunks of a message except
// the first one SHOULD use this type.
type Chunk3 struct {
	ChunkStreamID     byte
	ExtendedTimestamp uint32
	Body              []byte
}

// GetBody returns the chunk body.
func (c Chunk3) GetBody() []byte {
	return c.Body
}

// Read reads the chunk.
func (c *Chunk3) Read(r io.Reader, bodyLen uint32, hasExtendedTimestamp bool) error {
	header := make([]byte, 4)
	_, err := io.ReadFull(r, header[:1])
	if err != nil {
		return err
	}

	c.ChunkStreamID = header[0] & 0x3F
	c.ExtendedTimestamp = 0

	if hasExtendedTimestamp {
		_, err = io.ReadFull(r, header[:4])
		if err != nil {
			return err
		}
		c.ExtendedTimestamp = uint32(header[0])<<24 | uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3])
	}

	c.Body = make([]byte, bodyLen)
	_, err = io.ReadFull(r, c.Body)
	return err
}

func (c Chunk3) marshalSize(hasExtendedTimestamp bool) int {
	n := 1 + len(c.Body)
	if hasExtendedTimestamp {
		n += 4
	}
	return n
}

// MarshalHeaderTo writes the chunk header, that is the chunk without its body, into buf.
// buf must be large enough to contain the header (5 bytes are always enough).
// It returns the number of bytes written.
func (c Chunk3) MarshalHeaderTo(buf []byte, hasExtendedTimestamp bool) int {
	buf[0] = 3<<6 | c.ChunkStreamID

	if hasExtendedTimestamp {
		buf[1] = byte(c.ExtendedTimestamp >> 24)
		buf[2] = byte(c.ExtendedTimestamp >> 16)
		buf[3] = byte(c.ExtendedTimestamp >> 8)
		buf[4] = byte(c.ExtendedTimestamp)
		return 5
	}

	return 1
}

// Marshal writes the chunk.
//
// Deprecated: not used anymore.
func (c Chunk3) Marshal(hasExtendedTimestamp bool) ([]byte, error) {
	buf := make([]byte, c.marshalSize(hasExtendedTimestamp))
	n := c.MarshalHeaderTo(buf, hasExtendedTimestamp)
	copy(buf[n:], c.Body)
	return buf, nil
}

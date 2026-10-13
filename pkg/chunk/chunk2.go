package chunk

import (
	"io"
)

// Chunk2 is a type 2 chunk.
// Neither the stream ID nor the
// message length is included; this chunk has the same stream ID and
// message length as the preceding chunk.
type Chunk2 struct {
	ChunkStreamID  byte
	TimestampDelta uint32
	Body           []byte
}

// Read reads the chunk.
func (c *Chunk2) Read(r io.Reader, bodyLen uint32, _ bool) error {
	header := make([]byte, 4)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return err
	}

	c.ChunkStreamID = header[0] & 0x3F
	c.TimestampDelta = uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3])

	if c.TimestampDelta >= 0xFFFFFF {
		_, err = io.ReadFull(r, header[:4])
		if err != nil {
			return err
		}

		c.TimestampDelta = uint32(header[0])<<24 | uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3])
	}

	c.Body = make([]byte, bodyLen)
	_, err = io.ReadFull(r, c.Body)
	return err
}

func (c Chunk2) marshalSize() int {
	n := 4 + len(c.Body)
	if c.TimestampDelta >= 0xFFFFFF {
		n += 4
	}
	return n
}

// AppendHeader appends the chunk header, that is the chunk without its body, to buf.
func (c Chunk2) AppendHeader(buf []byte, _ bool) []byte {
	if c.TimestampDelta >= 0xFFFFFF {
		return append(buf,
			2<<6|c.ChunkStreamID,
			0xFF, 0xFF, 0xFF,
			byte(c.TimestampDelta>>24), byte(c.TimestampDelta>>16), byte(c.TimestampDelta>>8), byte(c.TimestampDelta))
	}

	return append(buf,
		2<<6|c.ChunkStreamID,
		byte(c.TimestampDelta>>16), byte(c.TimestampDelta>>8), byte(c.TimestampDelta))
}

// Marshal writes the chunk.
func (c Chunk2) Marshal(hasExtendedTimestamp bool) ([]byte, error) {
	buf := c.AppendHeader(make([]byte, 0, c.marshalSize()), hasExtendedTimestamp)
	return append(buf, c.Body...), nil
}

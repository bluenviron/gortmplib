package chunk

import (
	"io"
)

// Chunk1 is a type 1 chunk.
// The message stream ID is not
// included; this chunk takes the same stream ID as the preceding chunk.
// Streams with variable-sized messages (for example, many video
// formats) SHOULD use this format for the first chunk of each new
// message after the first.
type Chunk1 struct {
	ChunkStreamID  byte
	TimestampDelta uint32
	BodyLen        uint32
	Type           uint8
	Body           []byte
}

// GetBody returns the chunk body.
func (c Chunk1) GetBody() []byte {
	return c.Body
}

// Read reads the chunk.
func (c *Chunk1) Read(r io.Reader, maxBodyLen uint32, _ bool) error {
	header := make([]byte, 8)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return err
	}

	c.ChunkStreamID = header[0] & 0x3F
	c.TimestampDelta = uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3])
	c.BodyLen = uint32(header[4])<<16 | uint32(header[5])<<8 | uint32(header[6])
	c.Type = header[7]

	if c.TimestampDelta >= 0xFFFFFF {
		_, err = io.ReadFull(r, header[:4])
		if err != nil {
			return err
		}

		c.TimestampDelta = uint32(header[0])<<24 | uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3])
	}

	chunkBodyLen := min(c.BodyLen, maxBodyLen)

	c.Body = make([]byte, chunkBodyLen)
	_, err = io.ReadFull(r, c.Body)
	return err
}

func (c Chunk1) marshalSize() int {
	n := 8 + len(c.Body)
	if c.TimestampDelta >= 0xFFFFFF {
		n += 4
	}
	return n
}

// MarshalHeaderTo writes the chunk header, that is the chunk without its body, into buf.
// buf must be large enough to contain the header (12 bytes are always enough).
// It returns the number of bytes written.
func (c Chunk1) MarshalHeaderTo(buf []byte, _ bool) int {
	buf[0] = 1<<6 | c.ChunkStreamID
	buf[4] = byte(c.BodyLen >> 16)
	buf[5] = byte(c.BodyLen >> 8)
	buf[6] = byte(c.BodyLen)
	buf[7] = c.Type

	if c.TimestampDelta >= 0xFFFFFF {
		buf[1] = 0xFF
		buf[2] = 0xFF
		buf[3] = 0xFF
		buf[8] = byte(c.TimestampDelta >> 24)
		buf[9] = byte(c.TimestampDelta >> 16)
		buf[10] = byte(c.TimestampDelta >> 8)
		buf[11] = byte(c.TimestampDelta)
		return 12
	}

	buf[1] = byte(c.TimestampDelta >> 16)
	buf[2] = byte(c.TimestampDelta >> 8)
	buf[3] = byte(c.TimestampDelta)
	return 8
}

// Marshal writes the chunk.
//
// Deprecated: not used anymore.
func (c Chunk1) Marshal(_ bool) ([]byte, error) {
	buf := make([]byte, c.marshalSize())
	n := c.MarshalHeaderTo(buf, false)
	copy(buf[n:], c.Body)
	return buf, nil
}

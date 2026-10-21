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

// AppendHeader appends the chunk header, that is the chunk without its body, to buf.
func (c Chunk1) AppendHeader(buf []byte, _ bool) []byte {
	if c.TimestampDelta >= 0xFFFFFF {
		return append(buf,
			1<<6|c.ChunkStreamID,
			0xFF, 0xFF, 0xFF,
			byte(c.BodyLen>>16), byte(c.BodyLen>>8), byte(c.BodyLen),
			c.Type,
			byte(c.TimestampDelta>>24), byte(c.TimestampDelta>>16), byte(c.TimestampDelta>>8), byte(c.TimestampDelta))
	}

	return append(buf,
		1<<6|c.ChunkStreamID,
		byte(c.TimestampDelta>>16), byte(c.TimestampDelta>>8), byte(c.TimestampDelta),
		byte(c.BodyLen>>16), byte(c.BodyLen>>8), byte(c.BodyLen),
		c.Type)
}

// Marshal writes the chunk.
func (c Chunk1) Marshal(hasExtendedTimestamp bool) ([]byte, error) {
	buf := c.AppendHeader(make([]byte, 0, c.marshalSize()), hasExtendedTimestamp)
	return append(buf, c.Body...), nil
}

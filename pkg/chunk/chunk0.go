package chunk

import (
	"io"
)

// Chunk0 is a type 0 chunk.
// This type MUST be used at
// the start of a chunk stream, and whenever the stream timestamp goes
// backward (e.g., because of a backward seek).
type Chunk0 struct {
	ChunkStreamID   byte
	Timestamp       uint32
	BodyLen         uint32
	Type            uint8
	MessageStreamID uint32
	Body            []byte
}

// Read reads the chunk.
func (c *Chunk0) Read(r io.Reader, maxBodyLen uint32, _ bool) error {
	header := make([]byte, 12)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return err
	}

	c.ChunkStreamID = header[0] & 0x3F
	c.Timestamp = uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3])
	c.BodyLen = uint32(header[4])<<16 | uint32(header[5])<<8 | uint32(header[6])
	c.Type = header[7]
	c.MessageStreamID = uint32(header[8])<<24 | uint32(header[9])<<16 | uint32(header[10])<<8 | uint32(header[11])

	if c.Timestamp >= 0xFFFFFF {
		_, err = io.ReadFull(r, header[:4])
		if err != nil {
			return err
		}

		c.Timestamp = uint32(header[0])<<24 | uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3])
	}

	chunkBodyLen := min(c.BodyLen, maxBodyLen)

	c.Body = make([]byte, chunkBodyLen)
	_, err = io.ReadFull(r, c.Body)
	return err
}

func (c Chunk0) marshalSize() int {
	n := 12 + len(c.Body)
	if c.Timestamp >= 0xFFFFFF {
		n += 4
	}
	return n
}

// AppendHeader appends the chunk header, that is the chunk without its body, to buf.
func (c Chunk0) AppendHeader(buf []byte, _ bool) []byte {
	if c.Timestamp >= 0xFFFFFF {
		return append(buf,
			c.ChunkStreamID,
			0xFF, 0xFF, 0xFF,
			byte(c.BodyLen>>16), byte(c.BodyLen>>8), byte(c.BodyLen),
			c.Type,
			byte(c.MessageStreamID>>24), byte(c.MessageStreamID>>16), byte(c.MessageStreamID>>8), byte(c.MessageStreamID),
			byte(c.Timestamp>>24), byte(c.Timestamp>>16), byte(c.Timestamp>>8), byte(c.Timestamp))
	}

	return append(buf,
		c.ChunkStreamID,
		byte(c.Timestamp>>16), byte(c.Timestamp>>8), byte(c.Timestamp),
		byte(c.BodyLen>>16), byte(c.BodyLen>>8), byte(c.BodyLen),
		c.Type,
		byte(c.MessageStreamID>>24), byte(c.MessageStreamID>>16), byte(c.MessageStreamID>>8), byte(c.MessageStreamID))
}

// Marshal writes the chunk.
func (c Chunk0) Marshal(hasExtendedTimestamp bool) ([]byte, error) {
	buf := c.AppendHeader(make([]byte, 0, c.marshalSize()), hasExtendedTimestamp)
	return append(buf, c.Body...), nil
}

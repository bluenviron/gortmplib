package message //nolint:dupl

import (
	"fmt"

	"github.com/bluenviron/gortmplib/pkg/rawmessage"
)

// AbortMessage is an abort message.
type AbortMessage struct {
	ChunkStreamID uint32
}

func (m *AbortMessage) unmarshal(raw *rawmessage.Message) error {
	if len(raw.Body) != 4 {
		return fmt.Errorf("invalid body size")
	}

	m.ChunkStreamID = uint32(raw.Body[0])<<24 | uint32(raw.Body[1])<<16 | uint32(raw.Body[2])<<8 | uint32(raw.Body[3])

	return nil
}

func (m *AbortMessage) marshal() (*rawmessage.Message, error) {
	buf := make([]byte, 4)

	buf[0] = byte(m.ChunkStreamID >> 24)
	buf[1] = byte(m.ChunkStreamID >> 16)
	buf[2] = byte(m.ChunkStreamID >> 8)
	buf[3] = byte(m.ChunkStreamID)

	return &rawmessage.Message{
		ChunkStreamID: ControlChunkStreamID,
		Type:          uint8(TypeAbortMessage),
		Body:          buf,
	}, nil
}

// Package rawmessage contains a RTMP raw message reader/writer.
package rawmessage

import (
	"time"
)

// Message is a raw message.
type Message struct {
	ChunkStreamID   byte
	Timestamp       time.Duration
	Type            uint8
	MessageStreamID uint32
	Body            []byte

	// Payload is sent after Body, as part of the message body.
	// It allows to write a message made of a small header and a big payload
	// without copying them into a single buffer.
	// It is never filled by Reader.
	Payload []byte
}

func (m *Message) clone() *Message {
	return &Message{
		ChunkStreamID:   m.ChunkStreamID,
		Timestamp:       m.Timestamp,
		Type:            m.Type,
		MessageStreamID: m.MessageStreamID,
		Body:            m.Body,
		Payload:         m.Payload,
	}
}

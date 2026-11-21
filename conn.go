package gortmplib

import (
	"github.com/bluenviron/gortmplib/pkg/message"
)

// Conn is implemented by Client and ServerConn.
type Conn interface {
	BytesReceived() uint64
	BytesSent() uint64
	Read() (message.Message, error)
	// Write writes a message.
	// Writer reuses the buffers of a message after Write() has returned,
	// therefore Write() must not keep references to the message or to its fields.
	Write(msg message.Message) error
}

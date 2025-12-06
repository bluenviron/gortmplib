package gortmplib

import (
	"github.com/bluenviron/gortmplib/pkg/message"
)

// Conn is implemented by Client and ServerConn.
type Conn interface {
	BytesReceived() uint64
	BytesSent() uint64
	Read() (message.Message, error)
	Write(msg message.Message) error
}

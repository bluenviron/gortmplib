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

type RewindableConn struct {
	Conn Conn

	entries  []message.Message
	rewinded bool
}

func (r *RewindableConn) BytesReceived() uint64           { return r.Conn.BytesReceived() }
func (r *RewindableConn) BytesSent() uint64               { return r.Conn.BytesSent() }
func (r *RewindableConn) Write(msg message.Message) error { return r.Conn.Write(msg) }

func (r *RewindableConn) Read() (message.Message, error) {
	if !r.rewinded {
		msg, err := r.Conn.Read()
		if err == nil {
			r.entries = append(r.entries, msg)
		}
		return msg, err
	}

	if r.entries != nil {
		entry := r.entries[0]
		r.entries = r.entries[1:]

		if len(r.entries) == 0 {
			r.entries = nil // release entries from memory
		}

		return entry, nil
	}

	return r.Conn.Read()
}

func (r *RewindableConn) Rewind() {
	r.rewinded = true
}

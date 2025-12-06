package gortmplib

import "github.com/bluenviron/gortmplib/pkg/message"

type rewindableConn struct {
	Conn Conn

	entries  []message.Message
	rewinded bool
}

func (r *rewindableConn) BytesReceived() uint64           { return r.Conn.BytesReceived() }
func (r *rewindableConn) BytesSent() uint64               { return r.Conn.BytesSent() }
func (r *rewindableConn) Write(msg message.Message) error { return r.Conn.Write(msg) }

func (r *rewindableConn) Read() (message.Message, error) {
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

func (r *rewindableConn) Rewind() {
	r.rewinded = true
}

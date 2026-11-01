// Package rc4rw contains an RC4 ReadWriter.
package rc4rw

import (
	"crypto/rc4"
	"io"
)

// ReadWriter is an RC4 ReadWriter.
type ReadWriter struct {
	rw  io.ReadWriter
	in  *rc4.Cipher
	out *rc4.Cipher
}

// New creates a new RC4 ReadWriter with the given keys.
func New(rw io.ReadWriter, keyIn []byte, keyOut []byte) (*ReadWriter, error) {
	in, err := rc4.NewCipher(keyIn)
	if err != nil {
		return nil, err
	}

	out, err := rc4.NewCipher(keyOut)
	if err != nil {
		return nil, err
	}

	p := make([]byte, 1536)
	in.XORKeyStream(p, p)
	out.XORKeyStream(p, p)

	return &ReadWriter{
		rw:  rw,
		in:  in,
		out: out,
	}, nil
}

// Read implements io.Reader.
func (r *ReadWriter) Read(p []byte) (int, error) {
	n, err := r.rw.Read(p)
	if n == 0 {
		return 0, err
	}

	r.in.XORKeyStream(p[:n], p[:n])
	return n, err
}

// Write implements io.Writer.
func (r *ReadWriter) Write(p []byte) (int, error) {
	r.out.XORKeyStream(p, p)
	return r.rw.Write(p)
}

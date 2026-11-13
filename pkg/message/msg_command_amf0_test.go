package message_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/bluenviron/gortmplib/pkg/amf0"
	"github.com/bluenviron/gortmplib/pkg/bytecounter"
	"github.com/bluenviron/gortmplib/pkg/message"
)

var benchmarkCommandAMF0 = &message.CommandAMF0{
	ChunkStreamID: 3,
	Name:          "connect",
	CommandID:     1,
	Arguments: []any{
		amf0.Object{
			{Key: "app", Value: "/stream"},
			{Key: "flashVer", Value: "LNX 9,0,124,2"},
			{Key: "tcUrl", Value: "http://example.com"},
			{Key: "fpad", Value: false},
			{Key: "capabilities", Value: float64(15)},
			{Key: "audioCodecs", Value: float64(4071)},
			{Key: "videoCodecs", Value: float64(252)},
			{Key: "videoFunction", Value: float64(1)},
		},
	},
}

// cycleReader repeats a buffer forever.
type cycleReader struct {
	buf []byte
	pos int
}

func (r *cycleReader) Read(p []byte) (int, error) {
	n := copy(p, r.buf[r.pos:])
	r.pos = (r.pos + n) % len(r.buf)
	return n, nil
}

func BenchmarkWriterCommandAMF0(b *testing.B) {
	bcw := bytecounter.NewWriter(io.Discard)
	w := &message.Writer{BW: bcw, BCW: bcw}
	w.Initialize()

	for b.Loop() {
		err := w.Write(benchmarkCommandAMF0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReaderCommandAMF0(b *testing.B) {
	// every message is written with a full header, therefore the stream can be repeated.
	var buf bytes.Buffer
	bcw := bytecounter.NewWriter(&buf)
	w := &message.Writer{BW: bcw, BCW: bcw}
	w.Initialize()
	err := w.Write(benchmarkCommandAMF0)
	if err != nil {
		b.Fatal(err)
	}

	bcr := bytecounter.NewReader(&cycleReader{buf: buf.Bytes()})
	r := &message.Reader{BR: bcr, BCR: bcr}
	r.Initialize()

	for b.Loop() {
		_, err = r.Read()
		if err != nil {
			b.Fatal(err)
		}
	}
}

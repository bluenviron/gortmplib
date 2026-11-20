package chunk_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/gortmplib/pkg/chunk"
)

var cases = []struct {
	name                 string
	enc                  []byte
	bodyLen              int
	hasExtendedTimestamp bool
	dec                  chunk.Chunk
}{
	{
		"chunk0 standard",
		[]byte{
			0x19, 0xb1, 0xa1, 0x91, 0x0, 0x0, 0x14, 0x14,
			0x3, 0x5d, 0x17, 0x3d, 0x1, 0x2, 0x3, 0x4,
		},
		4,
		false,
		&chunk.Chunk0{
			ChunkStreamID:   25,
			Timestamp:       11641233,
			Type:            20,
			MessageStreamID: 56432445,
			BodyLen:         20,
			Body:            []byte{1, 2, 3, 4},
		},
	},
	{
		"chunk0 extended timestamp",
		[]byte{
			0x19, 0xff, 0xff, 0xff, 0x00, 0x00, 0x14, 0x0f,
			0x00, 0x31, 0x84, 0xb2, 0xff, 0x34, 0x86, 0xa2,
			0x05, 0x06, 0x07, 0x08,
		},
		4,
		false,
		&chunk.Chunk0{
			ChunkStreamID:   25,
			Timestamp:       0xFF3486a2,
			Type:            15,
			MessageStreamID: 3245234,
			BodyLen:         20,
			Body:            []byte{5, 6, 7, 8},
		},
	},
	{
		"chunk1 standard",
		[]byte{
			0x59, 0xb1, 0xa1, 0x91, 0x0, 0x0, 0x14, 0x14,
			0x1, 0x2, 0x3, 0x4,
		},
		4,
		false,
		&chunk.Chunk1{
			ChunkStreamID:  25,
			TimestampDelta: 11641233,
			Type:           20,
			BodyLen:        20,
			Body:           []byte{1, 2, 3, 4},
		},
	},
	{
		"chunk1 extended timestamp",
		[]byte{
			0x59, 0xff, 0xff, 0xff, 0x00, 0x00, 0x14, 0x14,
			0xff, 0x88, 0x4b, 0x6c, 0x05, 0x06, 0x07, 0x08,
		},
		4,
		false,
		&chunk.Chunk1{
			ChunkStreamID:  25,
			TimestampDelta: 0xFF884B6C,
			Type:           20,
			BodyLen:        20,
			Body:           []byte{5, 6, 7, 8},
		},
	},
	{
		"chunk2 standard",
		[]byte{
			0x99, 0xb1, 0xa1, 0x91, 0x1, 0x2, 0x3, 0x4,
		},
		4,
		false,
		&chunk.Chunk2{
			ChunkStreamID:  25,
			TimestampDelta: 11641233,
			Body:           []byte{1, 2, 3, 4},
		},
	},
	{
		"chunk2 extended timestamp",
		[]byte{
			0x99, 0xff, 0xff, 0xff, 0xff, 0xaa, 0xbb, 0xcc,
			0x05, 0x06, 0x07, 0x08,
		},
		4,
		false,
		&chunk.Chunk2{
			ChunkStreamID:  25,
			TimestampDelta: 0xFFAABBCC,
			Body:           []byte{5, 6, 7, 8},
		},
	},
	{
		"chunk3 standard",
		[]byte{
			0xd9, 0x1, 0x2, 0x3, 0x4,
		},
		4,
		false,
		&chunk.Chunk3{
			ChunkStreamID: 25,
			Body:          []byte{1, 2, 3, 4},
		},
	},
	{
		"chunk3 extended timestamp",
		[]byte{
			0xd9, 0xff, 0x34, 0x86, 0xa2, 0x05, 0x06, 0x07,
			0x08,
		},
		4,
		true,
		&chunk.Chunk3{
			ChunkStreamID:     25,
			ExtendedTimestamp: 0xFF3486A2,
			Body:              []byte{5, 6, 7, 8},
		},
	},
}

func TestChunkRead(t *testing.T) {
	for _, ca := range cases {
		t.Run(ca.name, func(t *testing.T) {
			chunk := reflect.New(reflect.TypeOf(ca.dec).Elem()).Interface().(chunk.Chunk)
			err := chunk.Read(bytes.NewReader(ca.enc), uint32(ca.bodyLen), ca.hasExtendedTimestamp)
			require.NoError(t, err)
			require.Equal(t, ca.dec, chunk)
			require.Equal(t, ca.bodyLen, len(ca.dec.GetBody()))
		})
	}
}

func TestChunkMarshal(t *testing.T) {
	for _, ca := range cases {
		t.Run(ca.name, func(t *testing.T) {
			buf, err := ca.dec.Marshal(ca.hasExtendedTimestamp)
			require.NoError(t, err)
			require.Equal(t, ca.enc, buf)
		})
	}
}

func TestChunkMarshalHeaderTo(t *testing.T) {
	for _, ca := range cases {
		t.Run(ca.name, func(t *testing.T) {
			// the buffer is dirty, in order to check that every header byte is written
			buf := bytes.Repeat([]byte{0xAA}, 16)
			var n int
			var body []byte

			switch ch := ca.dec.(type) {
			case *chunk.Chunk0:
				n, body = ch.MarshalHeaderTo(buf, ca.hasExtendedTimestamp), ch.Body
			case *chunk.Chunk1:
				n, body = ch.MarshalHeaderTo(buf, ca.hasExtendedTimestamp), ch.Body
			case *chunk.Chunk2:
				n, body = ch.MarshalHeaderTo(buf, ca.hasExtendedTimestamp), ch.Body
			case *chunk.Chunk3:
				n, body = ch.MarshalHeaderTo(buf, ca.hasExtendedTimestamp), ch.Body
			}

			require.Equal(t, ca.enc, append(buf[:n], body...))
		})
	}
}

func FuzzChunk0Read(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		var chunk chunk.Chunk0
		err := chunk.Read(bytes.NewReader(b), 65536, false)
		if err != nil {
			return
		}

		_, err = chunk.Marshal(false)
		require.NoError(t, err)
	})
}

func FuzzChunk1Read(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		var chunk chunk.Chunk1
		err := chunk.Read(bytes.NewReader(b), 65536, false)
		if err != nil {
			return
		}

		_, err = chunk.Marshal(false)
		require.NoError(t, err)
	})
}

func FuzzChunk2Read(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		var chunk chunk.Chunk2
		err := chunk.Read(bytes.NewReader(b), 65536, false)
		if err != nil {
			return
		}

		_, err = chunk.Marshal(false)
		require.NoError(t, err)
	})
}

func FuzzChunk3Read(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		var chunk chunk.Chunk3
		err := chunk.Read(bytes.NewReader(b), 65536, true)
		if err != nil {
			return
		}

		_, err = chunk.Marshal(false)
		require.NoError(t, err)
	})
}

// readKind describes how to build a chunk of a given type and its encoded header.
// bodyLen is the body length declared by the header (Chunk0, Chunk1)
// and also the value passed to Read (all types).
type readKind struct {
	name     string
	newChunk func() chunk.Chunk
	header   func(bodyLen uint32) []byte
}

func marshalHeader(c chunk.Chunk) []byte {
	buf := make([]byte, 16)
	return buf[:c.MarshalHeaderTo(buf, false)]
}

var readKinds = []readKind{
	{
		"Chunk0",
		func() chunk.Chunk { return &chunk.Chunk0{} },
		func(bodyLen uint32) []byte {
			return marshalHeader(&chunk.Chunk0{
				ChunkStreamID:   5,
				Timestamp:       100,
				BodyLen:         bodyLen,
				Type:            20,
				MessageStreamID: 1,
			})
		},
	},
	{
		"Chunk1",
		func() chunk.Chunk { return &chunk.Chunk1{} },
		func(bodyLen uint32) []byte {
			return marshalHeader(&chunk.Chunk1{
				ChunkStreamID:  5,
				TimestampDelta: 100,
				BodyLen:        bodyLen,
				Type:           20,
			})
		},
	},
	{
		"Chunk2",
		func() chunk.Chunk { return &chunk.Chunk2{} },
		func(uint32) []byte {
			return marshalHeader(&chunk.Chunk2{ChunkStreamID: 5, TimestampDelta: 100})
		},
	},
	{
		"Chunk3",
		func() chunk.Chunk { return &chunk.Chunk3{} },
		func(uint32) []byte {
			return marshalHeader(&chunk.Chunk3{ChunkStreamID: 5})
		},
	},
}

// dataThenErrReader returns its data and then err (io.EOF if nil), sticky.
// err is returned together with the last bytes, like some readers do.
type dataThenErrReader struct {
	data []byte
	err  error
}

func (r *dataThenErrReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]

	if len(r.data) != 0 {
		return n, nil
	}

	if r.err == nil {
		return n, io.EOF
	}
	return n, r.err
}

func declaredBodyLen(c chunk.Chunk) uint32 {
	switch ch := c.(type) {
	case *chunk.Chunk0:
		return ch.BodyLen
	case *chunk.Chunk1:
		return ch.BodyLen
	}
	return 0
}

// A header declaring a huge body must not make Read allocate it
// when the payload is missing or tiny.
// This test must not run in parallel since it measures global allocations.
func TestChunkReadHeaderOnlyAllocation(t *testing.T) {
	const declared = 10 * 1024 * 1024
	const maxAlloc = 1024 * 1024

	for _, kind := range readKinds {
		for _, pl := range []struct {
			name    string
			payload []byte
			wantErr error
		}{
			{"no payload", nil, io.EOF},
			{"tiny payload", bytes.Repeat([]byte{0x42}, 16), io.ErrUnexpectedEOF},
		} {
			t.Run(kind.name+" "+pl.name, func(t *testing.T) {
				r := bytes.NewReader(append(kind.header(declared), pl.payload...))
				ch := kind.newChunk()

				var before, after runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&before)
				err := ch.Read(r, declared, false)
				runtime.ReadMemStats(&after)

				require.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(maxAlloc))
				require.ErrorIs(t, err, pl.wantErr)
				require.True(t, bytes.Equal(pl.payload, ch.GetBody()), "body = %x, want %x", ch.GetBody(), pl.payload)
			})
		}
	}
}

func TestChunkReadBodyFraming(t *testing.T) {
	errSentinel := errors.New("sentinel")
	body := []byte{1, 2, 3, 4}
	next := []byte{0xAA, 0xBB, 0xCC}

	for _, bc := range []struct {
		name     string
		bodyLen  uint32
		payload  []byte // bytes after the header
		srcErr   error  // returned by the source after the payload
		wantErr  error
		wantBody []byte
		wantNext []byte // bytes that must be left unread
	}{
		{"complete body, next header untouched", 4, append(bytes.Clone(body), next...), nil, nil, body, next},
		{"zero length", 0, next, nil, nil, nil, next},
		{"partial body then EOF", 4, body[:2], nil, io.ErrUnexpectedEOF, body[:2], nil},
		{"no data then EOF", 4, nil, nil, io.EOF, nil, nil},
		{"partial body then sentinel", 4, body[:2], errSentinel, errSentinel, body[:2], nil},
		{"partial body then timeout", 4, body[:2], os.ErrDeadlineExceeded, os.ErrDeadlineExceeded, body[:2], nil},
		{"error on empty payload", 4, nil, errSentinel, errSentinel, nil, nil},
		{"final data with EOF", 4, body, io.EOF, nil, body, nil},
		{"final data with sentinel", 4, body, errSentinel, nil, body, nil},
	} {
		for _, kind := range readKinds {
			t.Run(kind.name+" "+bc.name, func(t *testing.T) {
				r := &dataThenErrReader{
					data: append(kind.header(bc.bodyLen), bc.payload...),
					err:  bc.srcErr,
				}
				ch := kind.newChunk()

				err := ch.Read(r, bc.bodyLen, false)
				require.ErrorIs(t, err, bc.wantErr)
				require.True(t, bytes.Equal(bc.wantBody, ch.GetBody()), "body = %x, want %x", ch.GetBody(), bc.wantBody)
				require.True(t, bytes.Equal(bc.wantNext, r.data), "unread = %x, want %x", r.data, bc.wantNext)
			})
		}
	}
}

func TestChunkReadBodyLenClamp(t *testing.T) {
	const declared = 1024 * 1024
	const maxBodyLen = 64 * 1024

	body := bytes.Repeat([]byte{0x42}, maxBodyLen)
	next := []byte{0xAA, 0xBB, 0xCC}

	for _, kind := range readKinds[:2] {
		t.Run(kind.name, func(t *testing.T) {
			r := &dataThenErrReader{data: append(append(kind.header(declared), body...), next...)}
			ch := kind.newChunk()

			err := ch.Read(r, maxBodyLen, false)
			require.NoError(t, err)
			require.Equal(t, uint32(declared), declaredBodyLen(ch))
			require.Equal(t, body, ch.GetBody())
			require.Equal(t, next, r.data)
		})
	}
}

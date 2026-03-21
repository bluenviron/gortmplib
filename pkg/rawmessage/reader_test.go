package rawmessage

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/gortmplib/pkg/bytecounter"
	"github.com/bluenviron/gortmplib/pkg/chunk"
)

var cases = []struct {
	name     string
	messages []*Message
	chunks   []chunk.Chunk
}{
	{
		"(chunk0) + (chunk1)",
		[]*Message{
			{
				ChunkStreamID:   27,
				Timestamp:       18576 * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x03}, 64),
			},
			{
				ChunkStreamID:   27,
				Timestamp:       (18576 + 15) * time.Millisecond,
				Type:            5,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x04}, 64),
			},
		},
		[]chunk.Chunk{
			&chunk.Chunk0{
				ChunkStreamID:   27,
				Timestamp:       18576,
				Type:            6,
				MessageStreamID: 3123,
				BodyLen:         64,
				Body:            bytes.Repeat([]byte{0x03}, 64),
			},
			&chunk.Chunk1{
				ChunkStreamID:  27,
				TimestampDelta: 15,
				Type:           5,
				BodyLen:        64,
				Body:           bytes.Repeat([]byte{0x04}, 64),
			},
		},
	},
	{
		"(chunk0) + (chunk2) + (chunk3)",
		[]*Message{
			{
				ChunkStreamID:   27,
				Timestamp:       18576 * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x03}, 64),
			},
			{
				ChunkStreamID:   27,
				Timestamp:       (18576 + 15) * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x04}, 64),
			},
			{
				ChunkStreamID:   27,
				Timestamp:       (18576 + 15 + 15) * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x05}, 64),
			},
		},
		[]chunk.Chunk{
			&chunk.Chunk0{
				ChunkStreamID:   27,
				Timestamp:       18576,
				Type:            6,
				MessageStreamID: 3123,
				BodyLen:         64,
				Body:            bytes.Repeat([]byte{0x03}, 64),
			},
			&chunk.Chunk2{
				ChunkStreamID:  27,
				TimestampDelta: 15,
				Body:           bytes.Repeat([]byte{0x04}, 64),
			},
			&chunk.Chunk3{
				ChunkStreamID: 27,
				Body:          bytes.Repeat([]byte{0x05}, 64),
			},
		},
	},
	{
		"(chunk0 + chunk3) + (chunk1 + chunk3) + (chunk2 + chunk3) + (chunk3 + chunk3)",
		[]*Message{
			{
				ChunkStreamID:   27,
				Timestamp:       18576 * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x03}, 190),
			},
			{
				ChunkStreamID:   27,
				Timestamp:       18576 * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x04}, 192),
			},
			{
				ChunkStreamID:   27,
				Timestamp:       (18576 + 15) * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x05}, 192),
			},
			{
				ChunkStreamID:   27,
				Timestamp:       (18576 + 15 + 15) * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x06}, 192),
			},
		},
		[]chunk.Chunk{
			&chunk.Chunk0{
				ChunkStreamID:   27,
				Timestamp:       18576,
				Type:            6,
				MessageStreamID: 3123,
				BodyLen:         190,
				Body:            bytes.Repeat([]byte{0x03}, 128),
			},
			&chunk.Chunk3{
				ChunkStreamID: 27,
				Body:          bytes.Repeat([]byte{0x03}, 62),
			},
			&chunk.Chunk1{
				ChunkStreamID:  27,
				TimestampDelta: 0,
				Type:           6,
				BodyLen:        192,
				Body:           bytes.Repeat([]byte{0x04}, 128),
			},
			&chunk.Chunk3{
				ChunkStreamID: 27,
				Body:          bytes.Repeat([]byte{0x04}, 64),
			},
			&chunk.Chunk2{
				ChunkStreamID:  27,
				TimestampDelta: 15,
				Body:           bytes.Repeat([]byte{0x05}, 128),
			},
			&chunk.Chunk3{
				ChunkStreamID: 27,
				Body:          bytes.Repeat([]byte{0x05}, 64),
			},
			&chunk.Chunk3{
				ChunkStreamID: 27,
				Body:          bytes.Repeat([]byte{0x06}, 128),
			},
			&chunk.Chunk3{
				ChunkStreamID: 27,
				Body:          bytes.Repeat([]byte{0x06}, 64),
			},
		},
	},
	{
		"(chunk0 + chunk3 with extended timestamp)",
		[]*Message{
			{
				ChunkStreamID:   27,
				Timestamp:       0xFF123456 * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{5}, 160),
			},
		},
		[]chunk.Chunk{
			&chunk.Chunk0{
				ChunkStreamID:   27,
				Timestamp:       4279383126,
				Type:            6,
				MessageStreamID: 3123,
				BodyLen:         160,
				Body:            bytes.Repeat([]byte{5}, 128),
			},
			&chunk.Chunk3{
				ChunkStreamID: 27,
				Body:          bytes.Repeat([]byte{5}, 32),
			},
		},
	},
	{
		"decreasing timestamp",
		[]*Message{
			{
				ChunkStreamID:   27,
				Timestamp:       16 * time.Second,
				Type:            6,
				MessageStreamID: 3123,
				Body:            []byte{1, 2},
			},
			{
				ChunkStreamID:   27,
				Timestamp:       17 * time.Second,
				Type:            6,
				MessageStreamID: 3123,
				Body:            []byte{3, 4},
			},
			{
				ChunkStreamID:   27,
				Timestamp:       16 * time.Second,
				Type:            6,
				MessageStreamID: 3123,
				Body:            []byte{5, 6},
			},
			{
				ChunkStreamID:   27,
				Timestamp:       17 * time.Second,
				Type:            6,
				MessageStreamID: 3123,
				Body:            []byte{7, 8},
			},
		},
		[]chunk.Chunk{
			&chunk.Chunk0{
				ChunkStreamID:   27,
				Timestamp:       16000,
				Type:            6,
				MessageStreamID: 3123,
				BodyLen:         2,
				Body:            []byte{1, 2},
			},
			&chunk.Chunk2{
				ChunkStreamID:  27,
				TimestampDelta: 1000,
				Body:           []byte{3, 4},
			},
			&chunk.Chunk0{
				ChunkStreamID:   27,
				Timestamp:       16000,
				Type:            6,
				MessageStreamID: 3123,
				BodyLen:         2,
				Body:            []byte{5, 6},
			},
			&chunk.Chunk2{
				ChunkStreamID:  27,
				TimestampDelta: 1000,
				Body:           []byte{7, 8},
			},
		},
	},
}

func TestReader(t *testing.T) {
	for _, ca := range cases {
		t.Run(ca.name, func(t *testing.T) {
			var buf bytes.Buffer
			br := bytecounter.NewReader(&buf)
			r := NewReader(br, br, func(_ uint32) error {
				return nil
			})

			hasExtendedTimestamp := false

			for _, cach := range ca.chunks {
				buf2, err := cach.Marshal(hasExtendedTimestamp)
				require.NoError(t, err)
				buf.Write(buf2)
				hasExtendedTimestamp = chunkHasExtendedTimestamp(cach)
			}

			var msgs []*Message

			for {
				msg, err := r.Read()
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				msgs = append(msgs, msg)
			}

			require.Equal(t, ca.messages, msgs)
		})
	}
}

func TestReaderAdditional(t *testing.T) {
	for _, ca := range []struct {
		name     string
		messages []*Message
		chunks   []chunk.Chunk
	}{
		{
			"(chunk0) + (chunk3)",
			[]*Message{
				{
					ChunkStreamID:   27,
					Timestamp:       4279383126000000,
					Type:            6,
					MessageStreamID: 3123,
					Body:            bytes.Repeat([]byte{5}, 15),
				},
				{
					ChunkStreamID:   27,
					Timestamp:       4279383126000000,
					Type:            6,
					MessageStreamID: 3123,
					Body:            bytes.Repeat([]byte{6}, 15),
				},
			},
			[]chunk.Chunk{
				&chunk.Chunk0{
					ChunkStreamID:   27,
					Timestamp:       4279383126,
					Type:            6,
					MessageStreamID: 3123,
					BodyLen:         15,
					Body:            bytes.Repeat([]byte{5}, 15),
				},
				&chunk.Chunk3{
					ChunkStreamID: 27,
					Body:          bytes.Repeat([]byte{6}, 15),
				},
			},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var buf bytes.Buffer
			br := bytecounter.NewReader(&buf)
			r := NewReader(br, br, func(_ uint32) error {
				return nil
			})

			hasExtendedTimestamp := false

			for _, cach := range ca.chunks {
				buf2, err := cach.Marshal(hasExtendedTimestamp)
				require.NoError(t, err)
				buf.Write(buf2)
				hasExtendedTimestamp = chunkHasExtendedTimestamp(cach)
			}

			var msgs []*Message

			for {
				msg, err := r.Read()
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				msgs = append(msgs, msg)
			}

			require.Equal(t, ca.messages, msgs)
		})
	}
}

func TestReaderAbortChunkStream(t *testing.T) {
	for _, ca := range []struct {
		name    string
		abortID uint32
	}{
		{
			name:    "unsupported low csid",
			abortID: 1,
		},
		{
			name:    "unsupported high csid",
			abortID: 64,
		},
		{
			name:    "known csid",
			abortID: 27,
		},
		{
			name:    "unknown csid",
			abortID: 28,
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var buf bytes.Buffer
			bc := bytecounter.NewReader(&buf)
			r := NewReader(bc, bc, func(_ uint32) error {
				return nil
			})
			_ = r.SetChunkSize(64)

			chunks := []chunk.Chunk{
				&chunk.Chunk0{
					ChunkStreamID:   27,
					Timestamp:       1000,
					Type:            6,
					MessageStreamID: 3123,
					BodyLen:         80,
					Body:            bytes.Repeat([]byte{0xaa}, 64),
				},
				&chunk.Chunk0{
					ChunkStreamID:   28,
					Timestamp:       2000,
					Type:            7,
					MessageStreamID: 4123,
					BodyLen:         4,
					Body:            []byte{1, 2, 3, 4},
				},
				&chunk.Chunk0{
					ChunkStreamID:   27,
					Timestamp:       3000,
					Type:            8,
					MessageStreamID: 3123,
					BodyLen:         3,
					Body:            []byte{9, 8, 7},
				},
			}

			hasExtendedTimestamp := false
			for _, cach := range chunks {
				buf2, err := cach.Marshal(hasExtendedTimestamp)
				require.NoError(t, err)
				buf.Write(buf2)
				hasExtendedTimestamp = chunkHasExtendedTimestamp(cach)
			}

			msg, err := r.Read()
			require.NoError(t, err)
			require.Equal(t, byte(28), msg.ChunkStreamID)
			require.Equal(t, []byte{1, 2, 3, 4}, msg.Body)

			r.AbortChunkStream(ca.abortID)

			msg, err = r.Read()
			if ca.abortID != 27 {
				require.EqualError(t, err, "received type 0 chunk but expected type 3 chunk")
				return
			}

			require.NoError(t, err)
			require.Equal(t, byte(27), msg.ChunkStreamID)
			require.Equal(t, []byte{9, 8, 7}, msg.Body)
		})
	}
}

func TestReaderAcknowledge(t *testing.T) {
	for _, ca := range []string{"standard", "overflow"} {
		t.Run(ca, func(t *testing.T) {
			onAckCalled := make(chan struct{})

			var buf bytes.Buffer
			bc := bytecounter.NewReader(&buf)
			r := NewReader(bc, bc, func(_ uint32) error {
				close(onAckCalled)
				return nil
			})

			if ca == "overflow" {
				bc.SetCount(4294967096)
				r.lastAckCount = 4294967096
			}

			err := r.SetChunkSize(65536)
			require.NoError(t, err)

			r.SetWindowAckSize(100)

			buf2, err := chunk.Chunk0{
				ChunkStreamID:   27,
				Timestamp:       18576,
				Type:            6,
				MessageStreamID: 3123,
				BodyLen:         200,
				Body:            bytes.Repeat([]byte{0x03}, 200),
			}.Marshal(false)
			require.NoError(t, err)
			buf.Write(buf2)

			_, err = r.Read()
			require.NoError(t, err)

			<-onAckCalled
		})
	}
}

func FuzzReader(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		bcr := bytecounter.NewReader(bytes.NewReader(b))
		r := NewReader(bcr, bcr, func(_ uint32) error {
			return nil
		})

		var buf bytes.Buffer
		bcw := bytecounter.NewWriter(&buf)
		w := NewWriter(bcw, bcw, true)

		for {
			msg, err := r.Read()
			if err == nil {
				err = w.Write(msg)
				require.NoError(t, err)
			} else {
				break
			}
		}
	})
}

package rawmessage_test

import (
	"bytes"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/gortmplib/pkg/bytecounter"
	"github.com/bluenviron/gortmplib/pkg/chunk"
	"github.com/bluenviron/gortmplib/pkg/rawmessage"
)

func chunkBodySize(ch chunk.Chunk) uint32 {
	switch ch := ch.(type) {
	case *chunk.Chunk0:
		return uint32(len(ch.Body))
	case *chunk.Chunk1:
		return uint32(len(ch.Body))
	case *chunk.Chunk2:
		return uint32(len(ch.Body))
	case *chunk.Chunk3:
		return uint32(len(ch.Body))
	}
	return 0
}

func chunkHasExtendedTimestamp(ch chunk.Chunk) bool {
	switch ch := ch.(type) {
	case *chunk.Chunk0:
		return ch.Timestamp >= 0xFFFFFF
	case *chunk.Chunk1:
		return ch.TimestampDelta >= 0xFFFFFF
	case *chunk.Chunk2:
		return ch.TimestampDelta >= 0xFFFFFF
	case *chunk.Chunk3:
		return false
	}
	return false
}

func TestWriter(t *testing.T) {
	for _, ca := range cases {
		t.Run(ca.name, func(t *testing.T) {
			var buf bytes.Buffer
			bc := bytecounter.NewWriter(&buf)
			w := &rawmessage.Writer{
				BW:               bc,
				BCW:              bc,
				CheckAcknowledge: true,
			}
			w.Initialize()

			for _, msg := range ca.messages {
				err := w.Write(msg)
				require.NoError(t, err)
			}

			hasExtendedTimestamp := false

			for _, cach := range ca.chunks {
				ch := reflect.New(reflect.TypeOf(cach).Elem()).Interface().(chunk.Chunk)
				err := ch.Read(&buf, chunkBodySize(cach), hasExtendedTimestamp)
				require.NoError(t, err)
				require.Equal(t, cach, ch)
				hasExtendedTimestamp = chunkHasExtendedTimestamp(cach)
			}

			require.Zero(t, buf.Len())
		})
	}
}

func TestWriterAcknowledge(t *testing.T) {
	for _, ca := range []string{"standard", "overflow"} {
		t.Run(ca, func(t *testing.T) {
			var buf bytes.Buffer
			bc := bytecounter.NewWriter(&buf)
			w := &rawmessage.Writer{
				BW:               bc,
				BCW:              bc,
				CheckAcknowledge: true,
			}
			w.Initialize()

			if ca == "overflow" {
				bc.SetCount(4294967096)
				w.AckValue = 4294967096
			}

			w.SetChunkSize(65536)
			w.SetWindowAckSize(100)

			err := w.Write(&rawmessage.Message{
				ChunkStreamID:   27,
				Timestamp:       18576 * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x03}, 200),
			})
			require.NoError(t, err)

			err = w.Write(&rawmessage.Message{
				ChunkStreamID:   27,
				Timestamp:       18576 * time.Millisecond,
				Type:            6,
				MessageStreamID: 3123,
				Body:            bytes.Repeat([]byte{0x03}, 200),
			})
			require.EqualError(t, err, "no acknowledge received within window")
		})
	}
}

func TestWriterConcurrent(t *testing.T) {
	const writers = 8
	const messagesPerWriter = 24

	var buf bytes.Buffer
	bcw := bytecounter.NewWriter(&buf)
	w := rawmessage.NewWriter(bcw, bcw, true)
	w.SetWindowAckSize(1 << 30)

	var wg sync.WaitGroup
	errs := make(chan error, writers)
	start := make(chan struct{})

	for id := range writers {
		wg.Go(func() {
			<-start

			for i := range messagesPerWriter {
				err := w.Write(&rawmessage.Message{
					ChunkStreamID:   byte(id%2 + 3),
					Timestamp:       0,
					Type:            9,
					MessageStreamID: uint32(id + 1),
					Body:            bytes.Repeat([]byte{byte(id), byte(i)}, 1024),
				})
				if err != nil {
					errs <- err
					return
				}
			}
		})
	}

	wg.Go(func() {
		<-start

		for range writers * messagesPerWriter {
			w.SetChunkSize(128)
			w.SetWindowAckSize(1 << 30)
			w.SetAcknowledgeValue(0)
		}
	})

	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	bcr := bytecounter.NewReader(bytes.NewReader(buf.Bytes()))
	r := rawmessage.NewReader(bcr, bcr, nil)
	seen := make(map[[2]int]bool)

	for range writers * messagesPerWriter {
		msg, err := r.Read()
		require.NoError(t, err)

		id := int(msg.MessageStreamID) - 1
		require.Len(t, msg.Body, 2048)
		i := int(msg.Body[1])
		require.GreaterOrEqual(t, id, 0)
		require.Less(t, id, writers)
		require.GreaterOrEqual(t, i, 0)
		require.Less(t, i, messagesPerWriter)
		require.Equal(t, byte(id%2+3), msg.ChunkStreamID)
		require.False(t, seen[[2]int{id, i}])
		seen[[2]int{id, i}] = true

		require.Zero(t, msg.Timestamp)
		require.Equal(t, uint8(9), msg.Type)
		require.Equal(t, uint32(id+1), msg.MessageStreamID)
		require.Equal(t, bytes.Repeat([]byte{byte(id), byte(i)}, 1024), msg.Body)
	}

	_, err := r.Read()
	require.ErrorIs(t, err, io.EOF)
}

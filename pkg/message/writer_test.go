package message_test

import (
	"bytes"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/gortmplib/pkg/bytecounter"
	"github.com/bluenviron/gortmplib/pkg/message"
)

func TestWriterConcurrentChunkSize(t *testing.T) {
	const messages = 100

	var buf bytes.Buffer
	bcw := bytecounter.NewWriter(&buf)
	w := &message.Writer{BW: bcw, BCW: bcw}
	w.Initialize()

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 2)

	wg.Go(func() {
		<-start
		for range messages {
			if err := w.Write(&message.SetChunkSize{Value: 65536}); err != nil {
				errs <- err
				return
			}
			if err := w.Write(&message.SetChunkSize{Value: 128}); err != nil {
				errs <- err
				return
			}
		}
	})

	wg.Go(func() {
		<-start
		for range messages {
			if err := w.Write(&message.Video{
				ChunkStreamID:   message.VideoChunkStreamID,
				MessageStreamID: 1,
				Codec:           message.CodecH264,
				IsKeyFrame:      true,
				Type:            message.VideoTypeAU,
				AU:              bytes.Repeat([]byte{1}, 1024),
			}); err != nil {
				errs <- err
				return
			}
		}
	})

	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	bcr := bytecounter.NewReader(bytes.NewReader(buf.Bytes()))
	r := message.NewReader(bcr, bcr, nil)
	var controlCount, mediaCount int
	for range 3 * messages {
		msg, err := r.Read()
		require.NoError(t, err)
		switch msg.(type) {
		case *message.SetChunkSize:
			controlCount++
		case *message.Video:
			mediaCount++
		default:
			t.Fatalf("unexpected message: %T", msg)
		}
	}
	require.Equal(t, 2*messages, controlCount)
	require.Equal(t, messages, mediaCount)
	_, err := r.Read()
	require.ErrorIs(t, err, io.EOF)
}

func TestWriter(t *testing.T) {
	for _, ca := range readWriterCases {
		t.Run(ca.name, func(t *testing.T) {
			var buf bytes.Buffer
			bc := bytecounter.NewWriter(&buf)
			r := &message.Writer{
				BW:               bc,
				BCW:              bc,
				CheckAcknowledge: true,
			}
			r.Initialize()
			err := r.Write(ca.dec)
			require.NoError(t, err)
			require.Equal(t, ca.enc, buf.Bytes())
		})
	}
}

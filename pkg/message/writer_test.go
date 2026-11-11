package message_test

import (
	"bytes"
	"io"
	"sync"
	"testing"

	"github.com/abema/go-mp4"
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
				PacketType:      message.VideoPacketTypeAU,
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

func TestWriterVideoExDefaultFrameType(t *testing.T) {
	// VideoFrameType 0 is reserved. When FrameType is not set,
	// SequenceStart is written as KeyFrame and coded frames as InterFrame.
	for _, ca := range []struct {
		name   string
		msg    message.Message
		header []byte
	}{
		{
			"sequence start",
			&message.VideoExSequenceStart{
				ChunkStreamID:   6,
				MessageStreamID: 0x1000000,
				FourCC:          message.FourCCVP9,
				VP9Config:       &mp4.VpcC{FullBox: mp4.FullBox{Version: 1}, CodecInitializationData: []uint8{}},
			},
			[]byte{0x90},
		},
		{
			"coded frames",
			&message.VideoExCodedFrames{
				ChunkStreamID:   6,
				MessageStreamID: 0x1000000,
				FourCC:          message.FourCCHEVC,
				Payload:         []byte{1},
			},
			[]byte{0xa1},
		},
		{
			"frames x",
			&message.VideoExFramesX{
				ChunkStreamID:   6,
				MessageStreamID: 0x1000000,
				FourCC:          message.FourCCHEVC,
				Payload:         []byte{1},
			},
			[]byte{0xa3},
		},
		{
			"multitrack",
			&message.VideoExMultitrack{
				TrackID: 1,
				Wrapped: &message.VideoExFramesX{
					ChunkStreamID:   6,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCHEVC,
					Payload:         []byte{1},
				},
			},
			[]byte{0xa6, 0x03},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var buf bytes.Buffer
			bc := bytecounter.NewWriter(&buf)
			w := &message.Writer{BW: bc, BCW: bc}
			w.Initialize()
			err := w.Write(ca.msg)
			require.NoError(t, err)

			// skip the 12-byte type 0 chunk header
			require.Equal(t, ca.header, buf.Bytes()[12:12+len(ca.header)])
		})
	}
}

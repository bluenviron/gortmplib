package message_test

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

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

// The Writer recycles the buffers of the messages it has written.
// Messages of different sizes and types written one after the other must
// not leak anything of each other.
func TestWriterReusedBuffers(t *testing.T) {
	var buf bytes.Buffer
	bcw := bytecounter.NewWriter(&buf)
	w := &message.Writer{BW: bcw, BCW: bcw}
	w.Initialize()

	payload := func(fill byte, size int) []byte {
		return bytes.Repeat([]byte{fill}, size)
	}

	msgs := []message.Message{
		&message.Video{
			ChunkStreamID:   message.VideoChunkStreamID,
			MessageStreamID: 1,
			Codec:           message.CodecH264,
			FrameType:       message.VideoFrameTypeKeyFrame,
			PacketType:      message.VideoPacketTypeAU,
			PTSDelta:        66 * time.Millisecond,
			AU:              payload(0xA1, 5000),
		},
		&message.Audio{
			ChunkStreamID:   message.AudioChunkStreamID,
			MessageStreamID: 1,
			Codec:           message.CodecMPEG4Audio,
			Rate:            message.AudioRate44100,
			Depth:           message.AudioDepth16,
			IsStereo:        true,
			AACType:         message.AudioAACTypeAU,
			AU:              payload(0xA2, 300),
		},
		&message.Video{
			ChunkStreamID:   message.VideoChunkStreamID,
			MessageStreamID: 1,
			Codec:           message.CodecH264,
			FrameType:       message.VideoFrameTypeInterFrame,
			PacketType:      message.VideoPacketTypeAU,
			AU:              payload(0xA3, 20),
		},
		&message.VideoExMultitrack{
			MultitrackType: 0,
			TrackID:        3,
			Wrapped: &message.VideoExCodedFrames{
				ChunkStreamID:   message.VideoChunkStreamID,
				MessageStreamID: 1,
				FrameType:       message.VideoFrameTypeInterFrame,
				FourCC:          message.FourCCAVC,
				PTSDelta:        33 * time.Millisecond,
				Payload:         payload(0xA4, 7000),
			},
		},
		&message.VideoExFramesX{
			ChunkStreamID:   message.VideoChunkStreamID,
			MessageStreamID: 1,
			FrameType:       message.VideoFrameTypeInterFrame,
			FourCC:          message.FourCCAVC,
			Payload:         payload(0xA5, 100),
		},
		&message.Video{
			ChunkStreamID:   message.VideoChunkStreamID,
			MessageStreamID: 1,
			Codec:           message.CodecH264,
			FrameType:       message.VideoFrameTypeKeyFrame,
			PacketType:      message.VideoPacketTypeAU,
			AU:              payload(0xA6, 9000),
		},
	}

	// the same messages once more, to write them on top of the recycled buffers.
	msgs = append(msgs, msgs...)

	for _, msg := range msgs {
		require.NoError(t, w.Write(msg))
	}

	bcr := bytecounter.NewReader(bytes.NewReader(buf.Bytes()))
	r := message.NewReader(bcr, bcr, nil)

	for _, msg := range msgs {
		read, err := r.Read()
		require.NoError(t, err)

		switch exp := msg.(type) {
		case *message.Video:
			got := read.(*message.Video)
			require.Equal(t, exp.FrameType, got.FrameType)
			require.Equal(t, exp.PTSDelta, got.PTSDelta)
			require.Equal(t, exp.AU, got.AU)

		case *message.Audio:
			got := read.(*message.Audio)
			require.Equal(t, exp.AACType, got.AACType)
			require.Equal(t, exp.AU, got.AU)

		case *message.VideoExMultitrack:
			got := read.(*message.VideoExMultitrack)
			require.Equal(t, exp.TrackID, got.TrackID)
			require.Equal(t, exp.Wrapped.(*message.VideoExCodedFrames).Payload,
				got.Wrapped.(*message.VideoExCodedFrames).Payload)
			require.Equal(t, exp.Wrapped.(*message.VideoExCodedFrames).PTSDelta,
				got.Wrapped.(*message.VideoExCodedFrames).PTSDelta)

		case *message.VideoExFramesX:
			got := read.(*message.VideoExFramesX)
			require.Equal(t, exp.Payload, got.Payload)
		}
	}

	_, err := r.Read()
	require.ErrorIs(t, err, io.EOF)
}

// The buffer pool is shared by all Writers. Writers that run at the same time
// must not see the buffers of each other.
func TestWritersConcurrentBufferReuse(t *testing.T) {
	const (
		writers  = 8
		messages = 200
	)

	bufs := make([]bytes.Buffer, writers)
	var wg sync.WaitGroup

	for i := range writers {
		wg.Go(func() {
			bcw := bytecounter.NewWriter(&bufs[i])
			w := &message.Writer{BW: bcw, BCW: bcw}
			w.Initialize()

			for j := range messages {
				err := w.Write(&message.Video{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 1,
					Codec:           message.CodecH264,
					FrameType:       message.VideoFrameTypeInterFrame,
					PacketType:      message.VideoPacketTypeAU,
					AU:              bytes.Repeat([]byte{byte(i + 1)}, 1000+j*10),
				})
				if err != nil {
					t.Error(err)
					return
				}
			}
		})
	}

	wg.Wait()

	for i := range writers {
		bcr := bytecounter.NewReader(bytes.NewReader(bufs[i].Bytes()))
		r := message.NewReader(bcr, bcr, nil)

		for j := range messages {
			msg, err := r.Read()
			require.NoError(t, err)
			require.Equal(t, bytes.Repeat([]byte{byte(i + 1)}, 1000+j*10), msg.(*message.Video).AU)
		}
	}
}

package message

import (
	"bytes"
	"fmt"
	"time"

	"github.com/abema/go-mp4"

	"github.com/bluenviron/gortmplib/pkg/rawmessage"
)

const (
	// VideoChunkStreamID is the chunk stream ID that is usually used to send Video{}
	VideoChunkStreamID = 6
)

// video codecs
const (
	CodecH264 = 7
	CodecH265 = 12 // unofficial
)

// VideoType is the type of a video message.
type VideoType uint8

// VideoType values.
const (
	VideoTypeConfig VideoType = 0
	VideoTypeAU     VideoType = 1
	VideoTypeEOS    VideoType = 2
)

// Video is a video message.
type Video struct {
	ChunkStreamID   byte
	DTS             time.Duration
	MessageStreamID uint32
	Codec           uint8
	IsKeyFrame      bool
	Type            VideoType
	PTSDelta        time.Duration
	HEVCConfig      *mp4.HvcC                    // Type = VideoTypeConfig, Codec = CodecH265
	AVCConfig       *mp4.AVCDecoderConfiguration // Type = VideoTypeConfig, Codec = CodecH264
	AU              []byte                       // Type = VideoTypeAU
}

func (m *Video) unmarshal(raw *rawmessage.Message) error {
	m.ChunkStreamID = raw.ChunkStreamID
	m.DTS = raw.Timestamp
	m.MessageStreamID = raw.MessageStreamID

	if len(raw.Body) < 5 {
		return fmt.Errorf("invalid body size")
	}

	m.IsKeyFrame = (raw.Body[0] >> 4) == 1

	m.Codec = raw.Body[0] & 0x0F
	switch m.Codec {
	case CodecH264, CodecH265:
	default:
		return fmt.Errorf("unsupported video codec: %d", m.Codec)
	}

	m.Type = VideoType(raw.Body[1])
	switch m.Type {
	case VideoTypeConfig, VideoTypeAU, VideoTypeEOS:
	default:
		return fmt.Errorf("unsupported video message type: %d", m.Type)
	}

	m.PTSDelta = time.Duration(uint32(raw.Body[2])<<16|uint32(raw.Body[3])<<8|uint32(raw.Body[4])) * time.Millisecond

	switch m.Type {
	case VideoTypeConfig:
		switch m.Codec {
		case CodecH264:
			m.AVCConfig = &mp4.AVCDecoderConfiguration{}
			m.AVCConfig.SetType(mp4.BoxTypeAvcC())
			_, err := mp4.Unmarshal(bytes.NewReader(raw.Body[5:]), uint64(len(raw.Body[5:])), m.AVCConfig, mp4.Context{})
			if err != nil {
				return fmt.Errorf("unable to parse H264 config: %w", err)
			}

		case CodecH265:
			m.HEVCConfig = &mp4.HvcC{}
			_, err := mp4.Unmarshal(bytes.NewReader(raw.Body[5:]), uint64(len(raw.Body[5:])), m.HEVCConfig, mp4.Context{})
			if err != nil {
				return fmt.Errorf("unable to parse H265 config: %w", err)
			}
		}

	case VideoTypeAU:
		m.AU = raw.Body[5:]
	}

	return nil
}

func (m Video) marshal() (*rawmessage.Message, error) {
	var bodyData []byte

	switch m.Type {
	case VideoTypeConfig:
		switch m.Codec {
		case CodecH264:
			var buf bytes.Buffer
			_, err := mp4.Marshal(&buf, m.AVCConfig, mp4.Context{})
			if err != nil {
				return nil, err
			}
			bodyData = buf.Bytes()

		case CodecH265:
			var buf bytes.Buffer
			_, err := mp4.Marshal(&buf, m.HEVCConfig, mp4.Context{})
			if err != nil {
				return nil, err
			}
			bodyData = buf.Bytes()
		}

	case VideoTypeAU:
		bodyData = m.AU
	}

	body := make([]byte, 5+len(bodyData))

	if m.IsKeyFrame {
		body[0] = 1 << 4
	} else {
		body[0] = 2 << 4
	}
	body[0] |= m.Codec
	body[1] = uint8(m.Type)

	tmp := uint32(m.PTSDelta / time.Millisecond)
	body[2] = uint8(tmp >> 16)
	body[3] = uint8(tmp >> 8)
	body[4] = uint8(tmp)

	copy(body[5:], bodyData)

	return &rawmessage.Message{
		ChunkStreamID:   m.ChunkStreamID,
		Timestamp:       m.DTS,
		Type:            uint8(TypeVideo),
		MessageStreamID: m.MessageStreamID,
		Body:            body,
	}, nil
}

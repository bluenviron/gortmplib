package message

import (
	"fmt"
	"time"

	"github.com/bluenviron/gortmplib/pkg/rawmessage"
)

const (
	// AudioChunkStreamID is the chunk stream ID that is usually used to send Audio{}
	AudioChunkStreamID = 4
)

// audio codecs
const (
	CodecMPEG1Audio = 2
	CodecLPCM       = 3
	CodecPCMA       = 7
	CodecPCMU       = 8
	CodecMPEG4Audio = 10
)

// AudioRate is the audio rate of an Audio.
type AudioRate uint8

// audio rates
const (
	AudioRate5512  AudioRate = 0
	AudioRate11025 AudioRate = 1
	AudioRate22050 AudioRate = 2
	AudioRate44100 AudioRate = 3
)

// AudioDepth is the audio depth of an Audio.
type AudioDepth uint8

// audio depths
const (
	AudioDepth8  AudioDepth = 0
	AudioDepth16 AudioDepth = 1
)

// AudioAACType is the AAC type of an Audio.
type AudioAACType uint8

// AudioAACType values.
const (
	AudioAACTypeConfig AudioAACType = 0
	AudioAACTypeAU     AudioAACType = 1
)

// Audio is an audio message.
type Audio struct {
	ChunkStreamID   byte
	DTS             time.Duration
	MessageStreamID uint32
	Codec           uint8
	Rate            AudioRate
	Depth           AudioDepth
	IsStereo        bool
	AACType         AudioAACType // only for CodecMPEG4Audio
	Payload         []byte
}

func (m *Audio) unmarshal(raw *rawmessage.Message) error {
	m.ChunkStreamID = raw.ChunkStreamID
	m.DTS = raw.Timestamp
	m.MessageStreamID = raw.MessageStreamID

	if len(raw.Body) < 2 {
		return fmt.Errorf("invalid body size")
	}

	m.Codec = raw.Body[0] >> 4
	switch m.Codec {
	case CodecMPEG4Audio, CodecMPEG1Audio, CodecPCMA, CodecPCMU, CodecLPCM:
	default:
		return fmt.Errorf("unsupported audio codec: %d", m.Codec)
	}

	m.Rate = AudioRate((raw.Body[0] >> 2) & 0x03)
	m.Depth = AudioDepth((raw.Body[0] >> 1) & 0x01)

	if (raw.Body[0] & 0x01) != 0 {
		m.IsStereo = true
	}

	if m.Codec == CodecMPEG4Audio {
		m.AACType = AudioAACType(raw.Body[1])
		switch m.AACType {
		case AudioAACTypeConfig, AudioAACTypeAU:
		default:
			return fmt.Errorf("unsupported audio message type: %d", m.AACType)
		}

		m.Payload = raw.Body[2:]
	} else {
		m.Payload = raw.Body[1:]
	}

	return nil
}

func (m Audio) marshalBodySize() int {
	if m.Codec == CodecMPEG4Audio {
		return 2 + len(m.Payload)
	}
	return 1 + len(m.Payload)
}

func (m Audio) marshal() (*rawmessage.Message, error) {
	body := make([]byte, m.marshalBodySize())

	body[0] = m.Codec<<4 | byte(m.Rate)<<2 | byte(m.Depth)<<1

	if m.IsStereo {
		body[0] |= 1
	}

	if m.Codec == CodecMPEG4Audio {
		body[1] = uint8(m.AACType)
		copy(body[2:], m.Payload)
	} else {
		copy(body[1:], m.Payload)
	}

	return &rawmessage.Message{
		ChunkStreamID:   m.ChunkStreamID,
		Timestamp:       m.DTS,
		Type:            uint8(TypeAudio),
		MessageStreamID: m.MessageStreamID,
		Body:            body,
	}, nil
}

package message

import (
	"bytes"
	"fmt"
	"time"

	"github.com/abema/go-mp4"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h265"

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

// VideoFrameType is the frame type of a video message.
type VideoFrameType uint8

// VideoFrameType values.
const (
	VideoFrameTypeKeyFrame   VideoFrameType = 1
	VideoFrameTypeInterFrame VideoFrameType = 2
	VideoFrameTypeCommand    VideoFrameType = 5
)

// VideoCommand is the command carried by a video message with FrameType = VideoFrameTypeCommand.
type VideoCommand uint8

// VideoCommand values.
const (
	VideoCommandStartSeek VideoCommand = 0
	VideoCommandEndSeek   VideoCommand = 1
)

// VideoPacketType is the type of a video message.
type VideoPacketType uint8

// VideoPacketType values.
const (
	VideoPacketTypeConfig VideoPacketType = 0
	VideoPacketTypeAU     VideoPacketType = 1
	VideoPacketTypeEOS    VideoPacketType = 2
)

// VideoType is the type of a video message.
//
// Deprecated: replaced by VideoPacketType.
type VideoType = VideoPacketType

// VideoType values.
const (
	VideoTypeConfig VideoType = VideoPacketTypeConfig
	VideoTypeAU     VideoType = VideoPacketTypeAU
	VideoTypeEOS    VideoType = VideoPacketTypeEOS
)

func h264FindParams(avcc *mp4.AVCDecoderConfiguration) ([]byte, []byte, error) {
	if len(avcc.SequenceParameterSets) > 1 || len(avcc.PictureParameterSets) > 1 {
		return nil, nil, fmt.Errorf("multiple H264 parameters are not supported")
	}

	if len(avcc.SequenceParameterSets) == 0 || len(avcc.SequenceParameterSets[0].NALUnit) == 0 ||
		len(avcc.PictureParameterSets) == 0 || len(avcc.PictureParameterSets[0].NALUnit) == 0 {
		return nil, nil, fmt.Errorf("H264 parameters not provided")
	}

	return avcc.SequenceParameterSets[0].NALUnit, avcc.PictureParameterSets[0].NALUnit, nil
}

func h265FindParams(hvcc *mp4.HvcC) ([]byte, []byte, []byte, error) {
	var vps []byte
	var sps []byte
	var pps []byte

	for _, arr := range hvcc.NaluArrays {
		switch h265.NALUType(arr.NaluType) {
		case h265.NALUType_VPS_NUT, h265.NALUType_SPS_NUT, h265.NALUType_PPS_NUT:
			if len(arr.Nalus) != 1 {
				return nil, nil, nil, fmt.Errorf("multiple H265 parameters are not supported")
			}

			if len(arr.Nalus[0].NALUnit) == 0 {
				return nil, nil, nil, fmt.Errorf("H265 parameter not provided")
			}

			switch h265.NALUType(arr.NaluType) {
			case h265.NALUType_VPS_NUT:
				if vps != nil {
					return nil, nil, nil, fmt.Errorf("multiple H265 VPS are not supported")
				}
				vps = arr.Nalus[0].NALUnit

			case h265.NALUType_SPS_NUT:
				if sps != nil {
					return nil, nil, nil, fmt.Errorf("multiple H265 SPS are not supported")
				}
				sps = arr.Nalus[0].NALUnit

			case h265.NALUType_PPS_NUT:
				if pps != nil {
					return nil, nil, nil, fmt.Errorf("multiple H265 PPS are not supported")
				}
				pps = arr.Nalus[0].NALUnit
			}
		}
	}

	if vps == nil || sps == nil || pps == nil {
		return nil, nil, nil, fmt.Errorf("H265 parameters not provided")
	}

	return vps, sps, pps, nil
}

// Video is a video message.
type Video struct {
	ChunkStreamID   byte
	DTS             time.Duration
	MessageStreamID uint32
	FrameType       VideoFrameType

	// only in case of FrameType = VideoFrameTypeCommand.
	// Command frames carry no other field.
	Command VideoCommand

	// only in case of FrameType = VideoFrameTypeKeyFrame or FrameType = VideoFrameTypeInterFrame.
	Codec      uint8
	PacketType VideoPacketType
	PTSDelta   time.Duration

	// only in case of PacketType = VideoPacketTypeConfig, Codec = CodecH265.
	// Guaranteed to contain non-empty VPS, SPS and PPS NALUs.
	HEVCConfig *mp4.HvcC

	// only in case of PacketType = VideoPacketTypeConfig, Codec = CodecH264.
	// Might be nil.
	// When non-nil, guaranteed to contain non-empty SPS and PPS NALUs.
	AVCConfig *mp4.AVCDecoderConfiguration

	// only in case of PacketType = VideoPacketTypeAU.
	AU []byte

	// Deprecated: replaced by FrameType.
	IsKeyFrame bool

	// Deprecated: replaced by PacketType.
	Type VideoType
}

func (m *Video) unmarshal(raw *rawmessage.Message) error {
	m.ChunkStreamID = raw.ChunkStreamID
	m.DTS = raw.Timestamp
	m.MessageStreamID = raw.MessageStreamID

	if len(raw.Body) < 2 {
		return fmt.Errorf("invalid body size")
	}

	switch VideoFrameType(raw.Body[0] >> 4) {
	case VideoFrameTypeKeyFrame:
		m.FrameType = VideoFrameTypeKeyFrame
		m.IsKeyFrame = true

	case VideoFrameTypeCommand:
		m.FrameType = VideoFrameTypeCommand

	default:
		m.FrameType = VideoFrameTypeInterFrame
	}

	if m.FrameType == VideoFrameTypeCommand {
		m.Command = VideoCommand(raw.Body[1])
		switch m.Command {
		case VideoCommandStartSeek, VideoCommandEndSeek:
		default:
			return fmt.Errorf("unsupported video command: %d", m.Command)
		}
	} else {
		m.Codec = raw.Body[0] & 0x0F
		switch m.Codec {
		case CodecH264, CodecH265:
		default:
			return fmt.Errorf("unsupported video codec: %d", m.Codec)
		}

		if len(raw.Body) < 5 {
			return fmt.Errorf("invalid body size")
		}

		m.PacketType = VideoPacketType(raw.Body[1])
		switch m.PacketType {
		case VideoPacketTypeConfig, VideoPacketTypeAU, VideoPacketTypeEOS:
		default:
			return fmt.Errorf("unsupported video message type: %d", m.PacketType)
		}
		m.Type = m.PacketType

		m.PTSDelta = time.Duration(int32(uint32(raw.Body[2])<<24|uint32(raw.Body[3])<<16|
			uint32(raw.Body[4])<<8)>>8) * time.Millisecond

		switch m.PacketType {
		case VideoPacketTypeConfig:
			switch m.Codec {
			case CodecH264:
				if len(raw.Body) > 5 {
					m.AVCConfig = &mp4.AVCDecoderConfiguration{}
					m.AVCConfig.SetType(mp4.BoxTypeAvcC())
					_, err := mp4.Unmarshal(bytes.NewReader(raw.Body[5:]), uint64(len(raw.Body[5:])), m.AVCConfig, mp4.Context{})
					if err != nil {
						return fmt.Errorf("unable to parse H264 config: %w", err)
					}

					_, _, err = h264FindParams(m.AVCConfig)
					if err != nil {
						return fmt.Errorf("unable to parse H264 config: %w", err)
					}
				}

			case CodecH265:
				m.HEVCConfig = &mp4.HvcC{}
				_, err := mp4.Unmarshal(bytes.NewReader(raw.Body[5:]), uint64(len(raw.Body[5:])), m.HEVCConfig, mp4.Context{})
				if err != nil {
					return fmt.Errorf("unable to parse H265 config: %w", err)
				}

				_, _, _, err = h265FindParams(m.HEVCConfig)
				if err != nil {
					return fmt.Errorf("unable to parse H265 config: %w", err)
				}
			}

		case VideoPacketTypeAU:
			if len(raw.Body) < 6 {
				return fmt.Errorf("invalid body size")
			}
			m.AU = raw.Body[5:]
		}
	}

	return nil
}

func (m Video) marshal() (*rawmessage.Message, error) {
	// support for the deprecated field IsKeyFrame
	frameType := m.FrameType
	if frameType == 0 {
		if m.IsKeyFrame {
			frameType = VideoFrameTypeKeyFrame
		} else {
			frameType = VideoFrameTypeInterFrame
		}
	}

	if frameType == VideoFrameTypeCommand {
		return &rawmessage.Message{
			ChunkStreamID:   m.ChunkStreamID,
			Timestamp:       m.DTS,
			Type:            uint8(TypeVideo),
			MessageStreamID: m.MessageStreamID,
			Body:            []byte{uint8(frameType) << 4, uint8(m.Command)},
		}, nil
	}

	var bodyData []byte

	// support for the deprecated field Type
	if m.Type != 0 {
		m.PacketType = m.Type
	}

	switch m.PacketType {
	case VideoPacketTypeConfig:
		switch m.Codec {
		case CodecH264:
			if m.AVCConfig != nil {
				var buf bytes.Buffer
				_, err := mp4.Marshal(&buf, m.AVCConfig, mp4.Context{})
				if err != nil {
					return nil, err
				}
				bodyData = buf.Bytes()
			}

		case CodecH265:
			var buf bytes.Buffer
			_, err := mp4.Marshal(&buf, m.HEVCConfig, mp4.Context{})
			if err != nil {
				return nil, err
			}
			bodyData = buf.Bytes()
		}

	case VideoPacketTypeAU:
		bodyData = m.AU
	}

	body := make([]byte, 5+len(bodyData))

	body[0] = uint8(frameType)<<4 | m.Codec
	body[1] = uint8(m.PacketType)

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

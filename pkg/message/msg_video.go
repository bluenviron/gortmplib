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

// VideoType is the type of a video message.
type VideoType uint8

// VideoType values.
const (
	VideoTypeConfig VideoType = 0
	VideoTypeAU     VideoType = 1
	VideoTypeEOS    VideoType = 2
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
	Codec           uint8
	IsKeyFrame      bool
	Type            VideoType
	PTSDelta        time.Duration

	// only in case of Type = VideoTypeConfig, Codec = CodecH265.
	// Guaranteed to contain non-empty VPS, SPS and PPS NALUs.
	HEVCConfig *mp4.HvcC

	// only in case of Type = VideoTypeConfig, Codec = CodecH264.
	// Might be nil.
	// When non-nil, guaranteed to contain non-empty SPS and PPS NALUs.
	AVCConfig *mp4.AVCDecoderConfiguration

	// only in case of Type = VideoTypeAU.
	AU []byte
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

	m.PTSDelta = time.Duration(int32(uint32(raw.Body[2])<<24|uint32(raw.Body[3])<<16|
		uint32(raw.Body[4])<<8)>>8) * time.Millisecond

	switch m.Type {
	case VideoTypeConfig:
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

	case VideoTypeAU:
		if len(raw.Body) < 6 {
			return fmt.Errorf("invalid body size")
		}
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

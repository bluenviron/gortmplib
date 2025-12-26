package codecs

import "github.com/bluenviron/gortmplib/pkg/message"

// LPCM is the LPCM codec.
type LPCM struct {
	BitDepth     int
	SampleRate   int
	ChannelCount int
}

// IsVideo returns whether the codec is a video one.
func (*LPCM) IsVideo() bool {
	return false
}

// ID returns the codec ID.
func (*LPCM) ID() uint32 {
	return uint32(message.CodecLPCM)
}

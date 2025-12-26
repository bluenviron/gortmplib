package codecs

import "github.com/bluenviron/gortmplib/pkg/message"

// H265 is the H265 codec.
type H265 struct {
	VPS []byte
	SPS []byte
	PPS []byte
}

// IsVideo returns whether the codec is a video one.
func (*H265) IsVideo() bool {
	return true
}

// ID returns the codec ID.
func (*H265) ID() uint32 {
	return uint32(message.FourCCHEVC)
}

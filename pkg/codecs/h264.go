package codecs

import "github.com/bluenviron/gortmplib/pkg/message"

// H264 is the H264 codec.
type H264 struct {
	SPS []byte
	PPS []byte
}

// IsVideo returns whether the codec is a video one.
func (*H264) IsVideo() bool {
	return true
}

// ID returns the codec ID.
func (*H264) ID() uint32 {
	return uint32(message.CodecH264)
}

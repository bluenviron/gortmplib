package codecs

import "github.com/bluenviron/gortmplib/pkg/message"

// AC3 is the AC-3 codec.
type AC3 struct {
	SampleRate   int
	ChannelCount int
}

// IsVideo returns whether the codec is a video one.
func (*AC3) IsVideo() bool {
	return false
}

// ID returns the codec ID.
func (*AC3) ID() uint32 {
	return uint32(message.FourCCAC3)
}

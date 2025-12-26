package codecs

import "github.com/bluenviron/gortmplib/pkg/message"

// Opus is the Opus codec.
type Opus struct {
	ChannelCount int
}

// IsVideo returns whether the codec is a video one.
func (*Opus) IsVideo() bool {
	return false
}

// ID returns the codec ID.
func (*Opus) ID() uint32 {
	return uint32(message.FourCCOpus)
}

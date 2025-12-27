package codecs

import "github.com/bluenviron/gortmplib/pkg/message"

// G711 is the G711 codec.
type G711 struct {
	MULaw        bool
	SampleRate   int
	ChannelCount int
}

// IsVideo returns whether the codec is a video one.
func (*G711) IsVideo() bool {
	return false
}

// ID returns the codec ID.
func (c *G711) ID() uint32 {
	if c.MULaw {
		return message.CodecPCMU
	}
	return message.CodecPCMA
}

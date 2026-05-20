package codecs

import (
	"github.com/bluenviron/gortmplib/pkg/message"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/opus"
)

// Opus is the Opus codec.
type Opus struct {
	IDHeader *opus.IDHeader
}

// IsVideo returns whether the codec is a video one.
func (*Opus) IsVideo() bool {
	return false
}

// ID returns the codec ID.
func (*Opus) ID() uint32 {
	return uint32(message.FourCCOpus)
}

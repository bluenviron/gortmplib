package codecs

import (
	"github.com/bluenviron/gortmplib/pkg/message"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/flac"
)

// FLAC is the FLAC codec.
type FLAC struct {
	StreamInfo *flac.StreamInfo
}

// IsVideo returns whether the codec is a video one.
func (*FLAC) IsVideo() bool {
	return false
}

// ID returns the codec ID.
func (*FLAC) ID() uint32 {
	return uint32(message.FourCCFLAC)
}

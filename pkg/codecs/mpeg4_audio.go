package codecs

import (
	"github.com/bluenviron/gortmplib/pkg/message"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
)

// MPEG4Audio is a MPEG-4 Audio codec.
type MPEG4Audio struct {
	Config *mpeg4audio.AudioSpecificConfig
}

// IsVideo returns whether the codec is a video one.
func (*MPEG4Audio) IsVideo() bool {
	return false
}

// ID returns the codec ID.
func (*MPEG4Audio) ID() uint32 {
	return uint32(message.CodecMPEG4Audio)
}

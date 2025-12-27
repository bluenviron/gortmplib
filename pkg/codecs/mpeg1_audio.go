package codecs

import "github.com/bluenviron/gortmplib/pkg/message"

// MPEG1Audio is a MPEG-1 Audio codec.
type MPEG1Audio struct {
	// in Go, empty structs share the same pointer,
	// therefore they cannot be used as map keys
	// or in equality operations. Prevent this.
	unused int //nolint:unused
}

// IsVideo returns whether the codec is a video one.
func (*MPEG1Audio) IsVideo() bool {
	return false
}

// ID returns the codec ID.
func (*MPEG1Audio) ID() uint32 {
	return uint32(message.CodecMPEG1Audio)
}

package codecs

import "github.com/bluenviron/gortmplib/pkg/message"

// VP9 is the VP9 codec.
type VP9 struct {
	// in Go, empty structs share the same pointer,
	// therefore they cannot be used as map keys
	// or in equality operations. Prevent this.
	unused int //nolint:unused
}

// IsVideo returns whether the codec is a video one.
func (*VP9) IsVideo() bool {
	return true
}

// ID returns the codec ID.
func (*VP9) ID() uint32 {
	return uint32(message.FourCCVP9)
}

package codecs

import "github.com/bluenviron/gortmplib/pkg/message"

// AV1 is the AV1 codec.
type AV1 struct {
	// in Go, empty structs share the same pointer,
	// therefore they cannot be used as map keys
	// or in equality operations. Prevent this.
	unused int //nolint:unused
}

// IsVideo returns whether the codec is a video one.
func (*AV1) IsVideo() bool {
	return true
}

// ID returns the codec ID.
func (*AV1) ID() uint32 {
	return uint32(message.FourCCAV1)
}

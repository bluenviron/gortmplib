package message

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVideoSignedCompositionTime(t *testing.T) {
	for _, ca := range []struct {
		ptsDelta time.Duration
		encoded  []byte
	}{
		{-8388608 * time.Millisecond, []byte{0x80, 0x00, 0x00}},
		{-34 * time.Millisecond, []byte{0xff, 0xff, 0xde}},
		{0, []byte{0x00, 0x00, 0x00}},
		{8388607 * time.Millisecond, []byte{0x7f, 0xff, 0xff}},
	} {
		t.Run(ca.ptsDelta.String(), func(t *testing.T) {
			original := Video{
				Codec:    CodecH264,
				Type:     VideoTypeAU,
				PTSDelta: ca.ptsDelta,
			}

			raw, err := original.marshal()
			require.NoError(t, err)
			require.Equal(t, ca.encoded, raw.Body[2:5])

			var decoded Video
			err = decoded.unmarshal(raw)
			require.NoError(t, err)
			require.Equal(t, ca.ptsDelta, decoded.PTSDelta)
		})
	}
}

func TestVideoExCodedFramesSignedCompositionTime(t *testing.T) {
	for _, fourCC := range []FourCC{FourCCAVC, FourCCHEVC} {
		t.Run(fmt.Sprintf("%08x", fourCC), func(t *testing.T) {
			original := VideoExCodedFrames{
				FourCC:   fourCC,
				PTSDelta: -34 * time.Millisecond,
			}

			raw, err := original.marshal()
			require.NoError(t, err)

			var decoded VideoExCodedFrames
			err = decoded.unmarshal(raw)
			require.NoError(t, err)
			require.Equal(t, original.PTSDelta, decoded.PTSDelta)
		})
	}
}

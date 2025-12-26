//go:build cgo

// Package main contains an example.
package main

import (
	"context"
	"log"
	"net/url"
	"time"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortmplib/pkg/codecs"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
)

// This example shows how to:
// 1. connect to a RTMP server, announce a MPEG-4 Audio (AAC) track.
// 2. generate dummy LPCM audio samples.
// 3. encode audio samples with MPEG-4 Audio (AAC).
// 4. send MPEG-4 Audio access units to the server.

// This example requires the FFmpeg libraries, that can be installed with this command:
// apt install -y libavcodec-dev gcc pkg-config

func multiplyAndDivide(v, m, d int64) int64 {
	secs := v / d
	dec := v % d
	return (secs*m + dec*m/d)
}

func main() {
	u, err := url.Parse("rtmp://127.0.0.1:1935/stream/test")
	if err != nil {
		panic(err)
	}

	c := &gortmplib.Client{
		URL:     u,
		Publish: true,
	}
	err = c.Initialize(context.Background())
	if err != nil {
		panic(err)
	}
	defer c.Close()

	track := &gortmplib.Track{
		Codec: &codecs.MPEG4Audio{
			Config: &mpeg4audio.AudioSpecificConfig{
				Type:         mpeg4audio.ObjectTypeAACLC,
				SampleRate:   44100,
				ChannelCount: 2,
			},
		},
	}

	c.NetConn().SetReadDeadline(time.Now().Add(10 * time.Second))

	w := &gortmplib.Writer{
		Conn:   c,
		Tracks: []*gortmplib.Track{track},
	}
	err = w.Initialize()
	if err != nil {
		panic(err)
	}

	// setup LPCM -> MPEG-4 Audio encoder
	mp4aEnc := &mp4aEncoder{}
	err = mp4aEnc.initialize()
	if err != nil {
		panic(err)
	}
	defer mp4aEnc.close()

	start := time.Now()
	prevPTS := int64(0)

	// setup a ticker to sleep between writings
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	c.NetConn().SetReadDeadline(time.Time{})

	for range ticker.C {
		// get current timestamp
		pts := multiplyAndDivide(int64(time.Since(start)), int64(44100), int64(time.Second))

		// generate dummy LPCM audio samples
		samples := createDummyAudio(pts, prevPTS)

		// encode samples with MPEG-4 Audio
		aus, outPTS, err := mp4aEnc.encode(samples)
		if err != nil {
			panic(err)
		}
		if aus == nil {
			continue
		}

		log.Printf("writing access units")

		for _, au := range aus {
			err = w.WriteMPEG4Audio(track, time.Duration(outPTS*int64(time.Second)/44100), au)
			if err != nil {
				panic(err)
			}

			outPTS += mpeg4audio.SamplesPerAccessUnit
		}

		prevPTS = pts
	}
}

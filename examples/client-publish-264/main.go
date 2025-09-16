//go:build cgo

// Package main contains an example.
package main

import (
	"context"
	"log"
	"net/url"
	"time"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
)

// This example shows how to:
// 1. connect to a RTMP server, announce a H264 track.
// 2. generate dummy RGBA images.
// 3. encode images with H264.
// 4. send H264 access units to the server.

// This example requires the FFmpeg libraries, that can be installed with this command:
// apt install -y libavcodec-dev libswscale-dev gcc pkg-config

func main() {
	u, err := url.Parse("rtmp://127.0.0.1:1935/stream")
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

	track := &format.H264{}

	c.NetConn().SetReadDeadline(time.Now().Add(10 * time.Second))

	w := &gortmplib.Writer{
		Conn:   c,
		Tracks: []format.Format{track},
	}
	err = w.Initialize()
	if err != nil {
		panic(err)
	}

	// setup RGBA -> H264 encoder
	h264enc := &h264Encoder{
		Width:  640,
		Height: 480,
		FPS:    5,
	}
	err = h264enc.initialize()
	if err != nil {
		panic(err)
	}
	defer h264enc.close()

	start := time.Now()

	// setup a ticker to sleep between frames
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	c.NetConn().SetReadDeadline(time.Time{})

	for range ticker.C {
		// get current timestamp
		pts := time.Since(start)

		// create a dummy image
		img := createDummyImage()

		// encode the image with H264
		au, pts, err := h264enc.encode(img, pts)
		if err != nil {
			panic(err)
		}

		// wait for a H264 access unit
		if au == nil {
			continue
		}

		log.Printf("writing access unit")

		err = w.WriteH264(track, pts, pts, au)
		if err != nil {
			panic(err)
		}
	}
}

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
// 1. connect to a RTMP server.
// 2. read all tracks on a path.

func main() {
	u, err := url.Parse("rtmp://127.0.0.1:1935/stream")
	if err != nil {
		panic(err)
	}

	c := &gortmplib.Client{
		URL:     u,
		Publish: false,
	}
	err = c.Initialize(context.Background())
	if err != nil {
		panic(err)
	}
	defer c.Close()

	c.NetConn().SetReadDeadline(time.Now().Add(10 * time.Second))

	r := &gortmplib.Reader{
		Conn: c,
	}
	err = r.Initialize()
	if err != nil {
		panic(err)
	}

	log.Printf("available tracks:")

	for _, track := range r.Tracks() {
		log.Printf("%T", track)

		switch track := track.(type) {
		case *format.AV1:
			r.OnDataAV1(track, func(pts time.Duration, tu [][]byte) {
				log.Printf("incoming AV1 data, pts=%v, len=%v", pts, len(tu))
			})

		case *format.VP9:
			r.OnDataVP9(track, func(pts time.Duration, frame []byte) {
				log.Printf("incoming VP9 data, pts=%v, len=%v", pts, len(frame))
			})

		case *format.H265:
			r.OnDataH265(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
				log.Printf("incoming H265 data, pts=%v, pts=%v, len=%v", pts, dts, len(au))
			})

		case *format.H264:
			r.OnDataH264(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
				log.Printf("incoming H264 data, pts=%v, dts=%v, len=%v", pts, dts, len(au))
			})

		case *format.Opus:
			r.OnDataOpus(track, func(pts time.Duration, packet []byte) {
				log.Printf("incoming Opus data, pts=%v, len=%v", pts, len(packet))
			})

		case *format.MPEG4Audio:
			r.OnDataMPEG4Audio(track, func(pts time.Duration, au []byte) {
				log.Printf("incoming MPEG-4 Audio data, pts=%v, len=%v", pts, len(au))
			})

		case *format.MPEG1Audio:
			r.OnDataMPEG1Audio(track, func(pts time.Duration, frame []byte) {
				log.Printf("incoming MPEG-1 Audio data, pts=%v, len=%v", pts, len(frame))
			})

		case *format.AC3:
			r.OnDataAC3(track, func(pts time.Duration, frame []byte) {
				log.Printf("incoming AC3 data, pts=%v, len=%v", pts, len(frame))
			})

		case *format.G711:
			r.OnDataG711(track, func(pts time.Duration, samples []byte) {
				log.Printf("incoming G711 data, pts=%v, len=%v", pts, len(samples))
			})

		case *format.LPCM:
			r.OnDataLPCM(track, func(pts time.Duration, samples []byte) {
				log.Printf("incoming LPCM data, pts=%v, len=%v", pts, len(samples))
			})
		}
	}

	for {
		c.NetConn().SetReadDeadline(time.Now().Add(10 * time.Second))
		err = r.Read()
		if err != nil {
			panic(err)
		}
	}
}

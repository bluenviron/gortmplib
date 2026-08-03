// Package main contains an example.
package main

import (
	"context"
	"log"
	"net/url"
	"time"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortmplib/pkg/codecs"
)

// This example shows how to:
// 1. connect to a RTMP server through a URL.
// 2. read all available tracks on the URL.
// 3. republish the tracks on another RTMP server through another URL.

func main() {
	// URL format is rtmp://user:pass@host:port/path#streamKey
	playURL, err := url.Parse("rtmp://127.0.0.1:1935/stream")
	if err != nil {
		panic(err)
	}

	player := &gortmplib.Client{
		URL:     playURL,
		Publish: false,
	}
	err = player.Initialize(context.Background())
	if err != nil {
		panic(err)
	}
	defer player.Close()

	player.NetConn().SetReadDeadline(time.Now().Add(10 * time.Second))

	reader := &gortmplib.Reader{
		Conn: player,
	}
	err = reader.Initialize()
	if err != nil {
		panic(err)
	}

	log.Printf("republishing %d tracks", len(reader.Tracks()))

	// URL format is rtmp://user:pass@host:port/path#streamKey
	publishURL, err := url.Parse("rtmp://127.0.0.1:1935/another-stream")
	if err != nil {
		panic(err)
	}

	publisher := &gortmplib.Client{
		URL:     publishURL,
		Publish: true,
	}
	err = publisher.Initialize(context.Background())
	if err != nil {
		panic(err)
	}
	defer publisher.Close()

	publisher.NetConn().SetReadDeadline(time.Now().Add(10 * time.Second))

	writer := &gortmplib.Writer{
		Conn:   publisher,
		Tracks: reader.Tracks(),
	}
	err = writer.Initialize()
	if err != nil {
		panic(err)
	}

	for _, track := range reader.Tracks() {
		switch track.Codec.(type) {
		case *codecs.AV1:
			reader.OnDataAV1(track, func(pts time.Duration, tu [][]byte) {
				log.Printf("routing AV1 data, pts=%v, len=%v", pts, len(tu))

				err2 := writer.WriteAV1(track, pts, tu)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.VP9:
			reader.OnDataVP9(track, func(pts time.Duration, frame []byte) {
				log.Printf("routing VP9 data, pts=%v, len=%v", pts, len(frame))

				err2 := writer.WriteVP9(track, pts, frame)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.H265:
			reader.OnDataH265(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
				log.Printf("routing H265 data, pts=%v, pts=%v, len=%v", pts, dts, len(au))

				err2 := writer.WriteH265(track, pts, dts, au)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.H264:
			reader.OnDataH264(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
				log.Printf("routing H264 data, pts=%v, dts=%v, len=%v", pts, dts, len(au))

				err2 := writer.WriteH264(track, pts, dts, au)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.Opus:
			reader.OnDataOpus(track, func(pts time.Duration, packet []byte) {
				log.Printf("routing Opus data, pts=%v, len=%v", pts, len(packet))

				err2 := writer.WriteOpus(track, pts, packet)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.FLAC:
			reader.OnDataFLAC(track, func(pts time.Duration, frame []byte) {
				log.Printf("routing FLAC data, pts=%v, len=%v", pts, len(frame))

				err2 := writer.WriteFLAC(track, pts, frame)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.MPEG4Audio:
			reader.OnDataMPEG4Audio(track, func(pts time.Duration, au []byte) {
				log.Printf("routing MPEG-4 Audio data, pts=%v, len=%v", pts, len(au))

				err2 := writer.WriteMPEG4Audio(track, pts, au)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.MPEG1Audio:
			reader.OnDataMPEG1Audio(track, func(pts time.Duration, frame []byte) {
				log.Printf("routing MPEG-1 Audio data, pts=%v, len=%v", pts, len(frame))

				err2 := writer.WriteMPEG1Audio(track, pts, frame)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.AC3:
			reader.OnDataAC3(track, func(pts time.Duration, frame []byte) {
				log.Printf("routing AC3 data, pts=%v, len=%v", pts, len(frame))

				err2 := writer.WriteAC3(track, pts, frame)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.G711:
			reader.OnDataG711(track, func(pts time.Duration, samples []byte) {
				log.Printf("routing G711 data, pts=%v, len=%v", pts, len(samples))

				err2 := writer.WriteG711(track, pts, samples)
				if err2 != nil {
					panic(err2)
				}
			})

		case *codecs.LPCM:
			reader.OnDataLPCM(track, func(pts time.Duration, samples []byte) {
				log.Printf("routing LPCM data, pts=%v, len=%v", pts, len(samples))

				err2 := writer.WriteLPCM(track, pts, samples)
				if err2 != nil {
					panic(err2)
				}
			})
		}
	}

	for {
		player.NetConn().SetReadDeadline(time.Now().Add(10 * time.Second))
		err = reader.Read()
		if err != nil {
			panic(err)
		}
	}
}

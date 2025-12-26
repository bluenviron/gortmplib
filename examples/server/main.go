// Package main contains an example.
package main

import (
	"fmt"
	"log"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortmplib/pkg/codecs"
)

// This example shows how to:
// 1. create a RTMP server
// 2. accept a stream from a reader.
// 3. broadcast the stream to readers.

var (
	mutex     sync.Mutex
	publisher *gortmplib.ServerConn
	tracks    []*gortmplib.Track
	readers   []*gortmplib.Writer
)

func handlePublisher(sc *gortmplib.ServerConn) error {
	sc.RW.(net.Conn).SetReadDeadline(time.Now().Add(10 * time.Second))

	r := &gortmplib.Reader{
		Conn: sc,
	}
	err := r.Initialize()
	if err != nil {
		return err
	}

	mutex.Lock()

	if publisher != nil {
		mutex.Unlock()
		return fmt.Errorf("someone is already publishing")
	}

	publisher = sc
	tracks = r.Tracks()

	mutex.Unlock()

	defer func() {
		mutex.Lock()
		defer mutex.Unlock()

		if publisher == sc {
			publisher = nil

			for _, reader := range readers {
				reader.Conn.(*gortmplib.ServerConn).RW.(net.Conn).Close()
			}
		}
	}()

	log.Printf("conn %v is publishing:", sc.RW.(net.Conn).RemoteAddr())

	for _, track := range r.Tracks() {
		log.Printf("%T", track)

		switch track.Codec.(type) {
		case *codecs.AV1:
			r.OnDataAV1(track, func(pts time.Duration, tu [][]byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteAV1(track, pts, tu) //nolint:errcheck
				}
			})

		case *codecs.VP9:
			r.OnDataVP9(track, func(pts time.Duration, frame []byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteVP9(track, pts, frame) //nolint:errcheck
				}
			})

		case *codecs.H265:
			r.OnDataH265(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteH265(track, pts, dts, au) //nolint:errcheck
				}
			})

		case *codecs.H264:
			r.OnDataH264(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteH264(track, pts, dts, au) //nolint:errcheck
				}
			})

		case *codecs.Opus:
			r.OnDataOpus(track, func(pts time.Duration, packet []byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteOpus(track, pts, packet) //nolint:errcheck
				}
			})

		case *codecs.MPEG4Audio:
			r.OnDataMPEG4Audio(track, func(pts time.Duration, au []byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteMPEG4Audio(track, pts, au) //nolint:errcheck
				}
			})

		case *codecs.MPEG1Audio:
			r.OnDataMPEG1Audio(track, func(pts time.Duration, frame []byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteMPEG1Audio(track, pts, frame) //nolint:errcheck
				}
			})

		case *codecs.AC3:
			r.OnDataAC3(track, func(pts time.Duration, frame []byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteAC3(track, pts, frame) //nolint:errcheck
				}
			})

		case *codecs.G711:
			r.OnDataG711(track, func(pts time.Duration, samples []byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteG711(track, pts, samples) //nolint:errcheck
				}
			})

		case *codecs.LPCM:
			r.OnDataLPCM(track, func(pts time.Duration, samples []byte) {
				mutex.Lock()
				defer mutex.Unlock()

				for _, reader := range readers {
					reader.WriteLPCM(track, pts, samples) //nolint:errcheck
				}
			})
		}
	}

	for {
		sc.RW.(net.Conn).SetReadDeadline(time.Now().Add(10 * time.Second))
		err = r.Read()
		if err != nil {
			return err
		}
	}
}

func handleReader(sc *gortmplib.ServerConn) error {
	mutex.Lock()

	if publisher == nil {
		mutex.Unlock()
		return fmt.Errorf("wants to read but there is no publisher")
	}

	sc.RW.(net.Conn).SetReadDeadline(time.Now().Add(10 * time.Second))

	w := &gortmplib.Writer{
		Conn:   sc,
		Tracks: tracks,
	}
	err := w.Initialize()
	if err != nil {
		return err
	}

	readers = append(readers, w)

	mutex.Unlock()

	log.Printf("conn %v is reading", sc.RW.(net.Conn).RemoteAddr())

	defer func() {
		mutex.Lock()
		defer mutex.Unlock()

		readers = slices.DeleteFunc(readers, func(el *gortmplib.Writer) bool {
			return (el == w)
		})
	}()

	sc.RW.(net.Conn).SetReadDeadline(time.Time{})

	for {
		buf := make([]byte, 1024)
		_, err = sc.RW.Read(buf)
		if err != nil {
			return err
		}
	}
}

func handleConnInner(conn net.Conn) error {
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	sc := &gortmplib.ServerConn{
		RW: conn,
	}
	err := sc.Initialize()
	if err != nil {
		return err
	}

	err = sc.Accept()
	if err != nil {
		return err
	}

	if sc.Publish {
		return handlePublisher(sc)
	}
	return handleReader(sc)
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	log.Printf("conn %v opened", conn.RemoteAddr())
	err := handleConnInner(conn)
	log.Printf("conn %v closed: %v", conn.RemoteAddr(), err)
}

func main() {
	ln, err := net.Listen("tcp", ":1935")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	log.Printf("listening on :1935")

	for {
		var conn net.Conn
		conn, err = ln.Accept()
		if err != nil {
			panic(err)
		}

		go handleConn(conn)
	}
}

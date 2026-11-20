// Package main contains an example.
package main

import (
	"fmt"
	"log"
	"maps"
	"net"
	"sync"
	"time"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortmplib/pkg/codecs"
)

// This example shows how to:
// 1. create a RTMP server
// 2. accept a stream from a publisher.
// 3. broadcast the stream to players.

const (
	readTimeout  = 10 * time.Second
	writeTimeout = 10 * time.Second
)

type player struct {
	conn   net.Conn
	writer *gortmplib.Writer
}

type server struct {
	mutex     sync.Mutex
	publisher *gortmplib.ServerConn
	tracks    []*gortmplib.Track
	players   map[*player]struct{}
}

func (s *server) snapshotPlayers() map[*player]struct{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return maps.Clone(s.players)
}

func (s *server) handlePublisher(conn net.Conn, sc *gortmplib.ServerConn) error {
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))

	r := &gortmplib.Reader{
		Conn: sc,
	}
	err := r.Initialize()
	if err != nil {
		return err
	}

	s.mutex.Lock()

	if s.publisher != nil {
		s.mutex.Unlock()
		return fmt.Errorf("someone is already publishing")
	}

	s.publisher = sc
	s.tracks = r.Tracks()
	s.players = make(map[*player]struct{})

	s.mutex.Unlock()

	defer func() {
		s.mutex.Lock()
		s.publisher = nil
		s.tracks = nil
		oldPlayers := s.players
		s.players = nil
		s.mutex.Unlock()

		for player := range oldPlayers {
			player.conn.Close()
		}
	}()

	log.Printf("conn %v is publishing:", conn.RemoteAddr())

	for _, track := range r.Tracks() {
		log.Printf("%T", track)

		switch track.Codec.(type) {
		case *codecs.AV1:
			r.OnDataAV1(track, func(pts time.Duration, tu [][]byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteAV1(track, pts, tu); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.VP9:
			r.OnDataVP9(track, func(pts time.Duration, frame []byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteVP9(track, pts, frame); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.H265:
			r.OnDataH265(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteH265(track, pts, dts, au); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.H264:
			r.OnDataH264(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteH264(track, pts, dts, au); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.Opus:
			r.OnDataOpus(track, func(pts time.Duration, packet []byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteOpus(track, pts, packet); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.FLAC:
			r.OnDataFLAC(track, func(pts time.Duration, frame []byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteFLAC(track, pts, frame); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.MPEG4Audio:
			r.OnDataMPEG4Audio(track, func(pts time.Duration, au []byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteMPEG4Audio(track, pts, au); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.MPEG1Audio:
			r.OnDataMPEG1Audio(track, func(pts time.Duration, frame []byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteMPEG1Audio(track, pts, frame); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.AC3:
			r.OnDataAC3(track, func(pts time.Duration, frame []byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteAC3(track, pts, frame); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.G711:
			r.OnDataG711(track, func(pts time.Duration, samples []byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteG711(track, pts, samples); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})

		case *codecs.LPCM:
			r.OnDataLPCM(track, func(pts time.Duration, samples []byte) {
				for player := range s.snapshotPlayers() {
					player.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					if err2 := player.writer.WriteLPCM(track, pts, samples); err2 != nil {
						log.Printf("conn %v write failed: %v", player.conn.RemoteAddr(), err2)
						player.conn.Close()
					}
				}
			})
		}
	}

	for {
		// Read() can both read and write (acknowledgements and ping replies).
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		conn.SetWriteDeadline(time.Now().Add(writeTimeout))

		err = r.Read()
		if err != nil {
			return err
		}
	}
}

func (s *server) handlePlayer(conn net.Conn, sc *gortmplib.ServerConn) error {
	s.mutex.Lock()
	currentPublisher, currentTracks := s.publisher, s.tracks
	s.mutex.Unlock()

	if currentPublisher == nil {
		return fmt.Errorf("wants to read but there is no publisher")
	}

	p := &player{
		conn: conn,
		writer: &gortmplib.Writer{
			Conn:   sc,
			Tracks: currentTracks,
		},
	}

	conn.SetReadDeadline(time.Time{})
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	err := p.writer.Initialize()
	if err != nil {
		return err
	}

	s.mutex.Lock()
	if s.publisher != currentPublisher {
		s.mutex.Unlock()
		return fmt.Errorf("publisher changed while initializing player")
	}
	s.players[p] = struct{}{}
	s.mutex.Unlock()

	log.Printf("conn %v is reading", conn.RemoteAddr())

	defer func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		delete(s.players, p)
	}()

	return p.writer.Wait()
}

func (s *server) handleConn(conn net.Conn) {
	log.Printf("conn %v opened", conn.RemoteAddr())
	err := s.handleConnInner(conn)
	log.Printf("conn %v closed: %v", conn.RemoteAddr(), err)
}

func (s *server) handleConnInner(conn net.Conn) error {
	defer conn.Close()

	sc := &gortmplib.ServerConn{RW: conn}

	conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	err := sc.Initialize()
	if err != nil {
		return err
	}

	conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	err = sc.AcceptConn()
	if err != nil {
		return err
	}

	if sc.Publish {
		return s.handlePublisher(conn, sc)
	}
	return s.handlePlayer(conn, sc)
}

func main() {
	ln, err := net.Listen("tcp", ":1935")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	log.Printf("listening on :1935")

	s := &server{}

	for {
		var conn net.Conn
		conn, err = ln.Accept()
		if err != nil {
			panic(err)
		}

		go s.handleConn(conn)
	}
}

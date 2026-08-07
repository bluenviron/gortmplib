package gortmplib

import (
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/gortmplib/pkg/amf0"
	"github.com/bluenviron/gortmplib/pkg/bytecounter"
	"github.com/bluenviron/gortmplib/pkg/handshake"
	"github.com/bluenviron/gortmplib/pkg/message"
)

func TestServerConn(t *testing.T) {
	for _, ca := range []string{
		"auth 1",
		"auth 2",
		"auth 3",
		"read",
		"publish",
	} {
		t.Run(ca, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:9121")
			require.NoError(t, err)
			defer ln.Close()

			done := make(chan struct{})

			go func() {
				defer close(done)

				nconn, err2 := ln.Accept()
				require.NoError(t, err2)
				defer nconn.Close()

				conn := &ServerConn{
					RW: nconn,
				}
				err2 = conn.Initialize()
				require.NoError(t, err2)

				if ca == "auth 1" || ca == "auth 2" || ca == "auth 3" {
					err2 = conn.AcceptIfCredentialsMatch("myuser", "mypass")
					switch ca {
					case "auth 1":
						require.Error(t, err2, "need auth")
						return
					case "auth 2":
						require.Error(t, err2, "need auth 2")
						return
					case "auth 3":
						require.NoError(t, err2)
					}
				} else {
					err2 = conn.Accept()
					require.NoError(t, err2)
				}

				require.Equal(t, &url.URL{
					Scheme:   "rtmp",
					Host:     "127.0.0.1:9121",
					Path:     "/stream",
					RawQuery: "key=val",
				}, conn.URL)
				require.Equal(t, (ca == "publish"), conn.Publish)
				require.Equal(t, "LNX 9,0,124,2", conn.FlashVer)
			}()

			conn, err := net.Dial("tcp", "127.0.0.1:9121")
			require.NoError(t, err)
			defer conn.Close()
			bc := bytecounter.NewReadWriter(conn)

			_, _, err = handshake.DoClient(bc, false, false)
			require.NoError(t, err)

			mrw := message.NewReadWriter(bc, bc, true)

			switch ca {
			case "auth 1": //nolint:dupl
				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "connect",
					CommandID:     1,
					Arguments: []any{
						amf0.Object{
							{Key: "app", Value: "stream?key=val"},
							{Key: "flashVer", Value: "LNX 9,0,124,2"},
							{Key: "tcUrl", Value: "rtmp://127.0.0.1:9121/stream?key=val"},
							{Key: "fpad", Value: false},
							{Key: "capabilities", Value: float64(15)},
							{Key: "audioCodecs", Value: float64(4071)},
							{Key: "videoCodecs", Value: float64(252)},
							{Key: "videoFunction", Value: float64(1)},
						},
					},
				})
				require.NoError(t, err)

				var msg message.Message
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "_error",
					CommandID:     1,
					Arguments: []any{
						nil,
						amf0.Object{
							{Key: "level", Value: "error"},
							{Key: "code", Value: "NetConnection.Connect.Rejected"},
							{Key: "description", Value: "code=403 need auth; authmod=adobe"},
						},
					},
				}, msg)

			case "auth 2": //nolint:dupl
				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "connect",
					CommandID:     1,
					Arguments: []any{
						amf0.Object{
							{Key: "app", Value: "stream?key=val?authmod=adobe&user=myuser"},
							{Key: "flashVer", Value: "LNX 9,0,124,2"},
							{Key: "tcUrl", Value: "rtmp://127.0.0.1:9121/stream?key=val?authmod=adobe&user=myuser"},
							{Key: "fpad", Value: false},
							{Key: "capabilities", Value: float64(15)},
							{Key: "audioCodecs", Value: float64(4071)},
							{Key: "videoCodecs", Value: float64(252)},
							{Key: "videoFunction", Value: float64(1)},
						},
					},
				})
				require.NoError(t, err)

				var msg message.Message
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "_error",
					CommandID:     1,
					Arguments: []any{
						nil,
						amf0.Object{
							{Key: "level", Value: "error"},
							{Key: "code", Value: "NetConnection.Connect.Rejected"},
							{Key: "description", Value: "authmod=adobe ?reason=needauth&user=myuser&salt=testsalt&challenge=testchallenge"},
						},
					},
				}, msg)

			case "auth 3":
				clientChallenge := uuid.New().String()
				response := authResponse("myuser", "mypass", serverSalt, "", serverChallenge, clientChallenge)

				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "connect",
					CommandID:     1,
					Arguments: []any{
						amf0.Object{
							{
								Key: "app",
								Value: fmt.Sprintf("stream?key=val?authmod=adobe&user=myuser&challenge=%s&response=%s",
									clientChallenge, response),
							},
							{Key: "flashVer", Value: "LNX 9,0,124,2"},
							{
								Key: "tcUrl",
								Value: fmt.Sprintf("rtmp://127.0.0.1:9121/stream?key=val?authmod=adobe&user=myuser&challenge=%s&response=%s",
									clientChallenge, response),
							},
							{Key: "fpad", Value: false},
							{Key: "capabilities", Value: float64(15)},
							{Key: "audioCodecs", Value: float64(4071)},
							{Key: "videoCodecs", Value: float64(252)},
							{Key: "videoFunction", Value: float64(1)},
						},
					},
				})
				require.NoError(t, err)

				var msg message.Message
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.SetWindowAckSize{
					Value: 2500000,
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.SetPeerBandwidth{
					Value: 2500000,
					Type:  2,
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.SetChunkSize{
					Value: 65536,
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "_result",
					CommandID:     1,
					Arguments: []any{
						amf0.Object{
							{Key: "fmsVer", Value: "LNX 9,0,124,2"},
							{Key: "capabilities", Value: float64(31)},
						},
						amf0.Object{
							{Key: "level", Value: "status"},
							{Key: "code", Value: "NetConnection.Connect.Success"},
							{Key: "description", Value: "Connection succeeded."},
							{Key: "objectEncoding", Value: float64(0)},
						},
					},
				}, msg)

				err = mrw.Write(&message.SetChunkSize{
					Value: 65536,
				})
				require.NoError(t, err)

				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "createStream",
					CommandID:     2,
					Arguments: []any{
						nil,
					},
				})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "_result",
					CommandID:     2,
					Arguments: []any{
						nil,
						float64(1),
					},
				}, msg)

				err = mrw.Write(&message.UserControlSetBufferLength{
					BufferLength: 0x64,
				})
				require.NoError(t, err)

				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Name:            "play",
					CommandID:       0,
					Arguments: []any{
						nil,
						"",
					},
				})
				require.NoError(t, err)

			case "read":
				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "connect",
					CommandID:     1,
					Arguments: []any{
						amf0.Object{
							{Key: "app", Value: "stream?key=val"},
							{Key: "flashVer", Value: "LNX 9,0,124,2"},
							{Key: "tcUrl", Value: "rtmp://127.0.0.1:9121/stream?key=val"},
							{Key: "fpad", Value: false},
							{Key: "capabilities", Value: float64(15)},
							{Key: "audioCodecs", Value: float64(4071)},
							{Key: "videoCodecs", Value: float64(252)},
							{Key: "videoFunction", Value: float64(1)},
						},
					},
				})
				require.NoError(t, err)

				var msg message.Message
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.SetWindowAckSize{
					Value: 2500000,
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.SetPeerBandwidth{
					Value: 2500000,
					Type:  2,
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.SetChunkSize{
					Value: 65536,
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "_result",
					CommandID:     1,
					Arguments: []any{
						amf0.Object{
							{Key: "fmsVer", Value: "LNX 9,0,124,2"},
							{Key: "capabilities", Value: float64(31)},
						},
						amf0.Object{
							{Key: "level", Value: "status"},
							{Key: "code", Value: "NetConnection.Connect.Success"},
							{Key: "description", Value: "Connection succeeded."},
							{Key: "objectEncoding", Value: float64(0)},
						},
					},
				}, msg)

				err = mrw.Write(&message.SetChunkSize{
					Value: 65536,
				})
				require.NoError(t, err)

				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "createStream",
					CommandID:     2,
					Arguments: []any{
						nil,
					},
				})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "_result",
					CommandID:     2,
					Arguments: []any{
						nil,
						float64(1),
					},
				}, msg)

				err = mrw.Write(&message.UserControlSetBufferLength{
					BufferLength: 0x64,
				})
				require.NoError(t, err)

				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Name:            "play",
					CommandID:       0,
					Arguments: []any{
						nil,
						"",
					},
				})
				require.NoError(t, err)

			case "publish":
				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "connect",
					CommandID:     1,
					Arguments: []any{
						amf0.Object{
							{Key: "app", Value: "stream?key=val"},
							{Key: "flashVer", Value: "LNX 9,0,124,2"},
							{Key: "tcUrl", Value: "rtmp://127.0.0.1:9121/stream?key=val"},
							{Key: "fpad", Value: false},
							{Key: "capabilities", Value: float64(15)},
							{Key: "audioCodecs", Value: float64(4071)},
							{Key: "videoCodecs", Value: float64(252)},
							{Key: "videoFunction", Value: float64(1)},
						},
					},
				})
				require.NoError(t, err)

				var msg message.Message
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.SetWindowAckSize{
					Value: 2500000,
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.SetPeerBandwidth{
					Value: 2500000,
					Type:  2,
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.SetChunkSize{
					Value: 65536,
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "_result",
					CommandID:     1,
					Arguments: []any{
						amf0.Object{
							{Key: "fmsVer", Value: "LNX 9,0,124,2"},
							{Key: "capabilities", Value: float64(31)},
						},
						amf0.Object{
							{Key: "level", Value: "status"},
							{Key: "code", Value: "NetConnection.Connect.Success"},
							{Key: "description", Value: "Connection succeeded."},
							{Key: "objectEncoding", Value: float64(0)},
						},
					},
				}, msg)

				err = mrw.Write(&message.SetChunkSize{
					Value: 65536,
				})
				require.NoError(t, err)

				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "releaseStream",
					CommandID:     2,
					Arguments: []any{
						nil,
						"",
					},
				})
				require.NoError(t, err)

				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "FCPublish",
					CommandID:     3,
					Arguments: []any{
						nil,
						"",
					},
				})
				require.NoError(t, err)

				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "createStream",
					CommandID:     4,
					Arguments: []any{
						nil,
					},
				})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "_result",
					CommandID:     4,
					Arguments: []any{
						nil,
						float64(1),
					},
				}, msg)

				err = mrw.Write(&message.CommandAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Name:            "publish",
					CommandID:       5,
					Arguments: []any{
						nil,
						"",
						"live",
					},
				})
				require.NoError(t, err)
			}

			<-done
		})
	}
}

func TestServerConnURL(t *testing.T) {
	for _, ca := range []struct {
		name        string
		tcurl       string
		app         string
		streamKey   string
		expectedURL string
	}{
		{
			name:        "ffmpeg, publish, single-component path",
			tcurl:       "rtmp://localhost:1935/comp1",
			app:         "comp1",
			streamKey:   "",
			expectedURL: "rtmp://localhost:1935/comp1",
		},
		{
			name:        "ffmpeg, publish, single-component path, with query",
			tcurl:       "rtmp://localhost:1935/comp1?key=val",
			app:         "comp1?key=val",
			streamKey:   "",
			expectedURL: "rtmp://localhost:1935/comp1?key=val",
		},
		{
			name:        "ffmpeg, publish, two-component path",
			tcurl:       "rtmp://localhost:1935/comp1",
			app:         "comp1",
			streamKey:   "comp2",
			expectedURL: "rtmp://localhost:1935/comp1/comp2",
		},
		{
			name:        "ffmpeg, publish, two-component path, with query",
			tcurl:       "rtmp://localhost:1935/comp1",
			app:         "comp1",
			streamKey:   "comp2?key=val",
			expectedURL: "rtmp://localhost:1935/comp1/comp2?key=val",
		},
		{
			name:        "gstreamer, publish, rtmpsink, single-component path",
			tcurl:       "rtmp://localhost:1935/comp1",
			app:         "comp1",
			streamKey:   "",
			expectedURL: "rtmp://localhost:1935/comp1",
		},
		{
			name:        "gstreamer, publish, rtmpsink, single-component path, with query",
			tcurl:       "rtmp://localhost/comp1?key=val",
			app:         "comp1?key=val",
			streamKey:   "",
			expectedURL: "rtmp://localhost/comp1?key=val",
		},
		{
			name:        "gstreamer, publish, rtmpsink, two-component path",
			tcurl:       "rtmp://localhost/comp1",
			app:         "comp1",
			streamKey:   "comp2",
			expectedURL: "rtmp://localhost/comp1/comp2",
		},
		{
			name:        "gstreamer, publish, rtmpsink, two-component path, with query",
			tcurl:       "rtmp://localhost/comp1",
			app:         "comp1",
			streamKey:   "comp2?key=val",
			expectedURL: "rtmp://localhost/comp1/comp2?key=val",
		},
		{
			name:        "OBS, single-component path",
			tcurl:       "rtmp://localhost:1935/comp1",
			app:         "comp1",
			streamKey:   "",
			expectedURL: "rtmp://localhost:1935/comp1",
		},
		{
			name:        "OBS, single-component path, with query",
			tcurl:       "rtmp://localhost:1935/comp1?key=val&tee=taa",
			app:         "comp1?key=val&tee=taa",
			streamKey:   "",
			expectedURL: "rtmp://localhost:1935/comp1?key=val&tee=taa",
		},
		{
			name:        "OBS, two-component path",
			tcurl:       "rtmp://localhost:1935/comp1/comp2",
			app:         "comp1/comp2",
			streamKey:   "",
			expectedURL: "rtmp://localhost:1935/comp1/comp2",
		},
		{
			name:        "OBS, two-component path, with query",
			tcurl:       "rtmp://localhost:1935/comp1/comp2?key=val",
			app:         "comp1/comp2?key=val",
			streamKey:   "",
			expectedURL: "rtmp://localhost:1935/comp1/comp2?key=val",
		},
		{
			name:        "OBS, multi-rendition, single-component path",
			tcurl:       "rtmp://localhost/comp1",
			app:         "comp1",
			streamKey:   "",
			expectedURL: "rtmp://localhost/comp1",
		},
		{
			name:        "OBS, multi-rendition, two-component path",
			tcurl:       "rtmp://localhost:1935/comp1/comp2",
			app:         "comp1/comp2",
			streamKey:   "",
			expectedURL: "rtmp://localhost:1935/comp1/comp2",
		},
		{
			name:        "OBS, multi-rendition, two-component path, with query",
			tcurl:       "rtmp://localhost:1935/comp1/comp2?key=val",
			app:         "comp1/comp2?key=val",
			streamKey:   "?key=val",
			expectedURL: "rtmp://localhost:1935/comp1/comp2?key=val",
		},
		{
			name:        "Neko",
			tcurl:       "'rtmp://localhost:1935/stream",
			app:         "stream",
			streamKey:   "",
			expectedURL: "rtmp://localhost:1935/stream",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:9121")
			require.NoError(t, err)
			defer ln.Close()

			done := make(chan struct{})

			go func() {
				defer close(done)

				nconn, err2 := ln.Accept()
				require.NoError(t, err2)
				defer nconn.Close()

				conn := &ServerConn{
					RW: nconn,
				}
				err2 = conn.Initialize()
				require.NoError(t, err2)

				err2 = conn.Accept()
				require.NoError(t, err2)

				require.Equal(t, ca.expectedURL, conn.URL.String())
			}()

			conn, err := net.Dial("tcp", "127.0.0.1:9121")
			require.NoError(t, err)
			defer conn.Close()
			bc := bytecounter.NewReadWriter(conn)

			_, _, err = handshake.DoClient(bc, false, false)
			require.NoError(t, err)

			mrw := message.NewReadWriter(bc, bc, true)

			err = mrw.Write(&message.CommandAMF0{
				ChunkStreamID: 3,
				Name:          "connect",
				CommandID:     1,
				Arguments: []any{
					amf0.Object{
						{Key: "app", Value: ca.app},
						{Key: "flashVer", Value: "LNX 9,0,124,2"},
						{Key: "tcUrl", Value: ca.tcurl},
						{Key: "fpad", Value: false},
						{Key: "capabilities", Value: float64(15)},
						{Key: "audioCodecs", Value: float64(4071)},
						{Key: "videoCodecs", Value: float64(252)},
						{Key: "videoFunction", Value: float64(1)},
					},
				},
			})
			require.NoError(t, err)

			msg, err := mrw.Read()
			require.NoError(t, err)
			require.Equal(t, &message.SetWindowAckSize{
				Value: 2500000,
			}, msg)

			msg, err = mrw.Read()
			require.NoError(t, err)
			require.Equal(t, &message.SetPeerBandwidth{
				Value: 2500000,
				Type:  2,
			}, msg)

			msg, err = mrw.Read()
			require.NoError(t, err)
			require.Equal(t, &message.SetChunkSize{
				Value: 65536,
			}, msg)

			msg, err = mrw.Read()
			require.NoError(t, err)
			require.Equal(t, &message.CommandAMF0{
				ChunkStreamID: 3,
				Name:          "_result",
				CommandID:     1,
				Arguments: []any{
					amf0.Object{
						{Key: "fmsVer", Value: "LNX 9,0,124,2"},
						{Key: "capabilities", Value: float64(31)},
					},
					amf0.Object{
						{Key: "level", Value: "status"},
						{Key: "code", Value: "NetConnection.Connect.Success"},
						{Key: "description", Value: "Connection succeeded."},
						{Key: "objectEncoding", Value: float64(0)},
					},
				},
			}, msg)

			err = mrw.Write(&message.SetChunkSize{
				Value: 65536,
			})
			require.NoError(t, err)

			err = mrw.Write(&message.CommandAMF0{
				ChunkStreamID: 3,
				Name:          "createStream",
				CommandID:     2,
				Arguments: []any{
					nil,
				},
			})
			require.NoError(t, err)

			msg, err = mrw.Read()
			require.NoError(t, err)
			require.Equal(t, &message.CommandAMF0{
				ChunkStreamID: 3,
				Name:          "_result",
				CommandID:     2,
				Arguments: []any{
					nil,
					float64(1),
				},
			}, msg)

			err = mrw.Write(&message.UserControlSetBufferLength{
				BufferLength: 0x64,
			})
			require.NoError(t, err)

			err = mrw.Write(&message.CommandAMF0{
				ChunkStreamID:   4,
				MessageStreamID: 0x1000000,
				Name:            "play",
				CommandID:       0,
				Arguments: []any{
					nil,
					ca.streamKey,
				},
			})
			require.NoError(t, err)

			<-done
		})
	}
}

func TestServerConnFourCcList(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:9121")
	require.NoError(t, err)
	defer ln.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)

		nconn, err2 := ln.Accept()
		require.NoError(t, err2)
		defer nconn.Close()

		conn := &ServerConn{
			RW: nconn,
		}
		err2 = conn.Initialize()
		require.NoError(t, err2)

		require.Equal(t, amf0.StrictArray{
			"av01",
			"Avc1",
		}, conn.FourCcList)
	}()

	conn, err := net.Dial("tcp", "127.0.0.1:9121")
	require.NoError(t, err)
	defer conn.Close()
	bc := bytecounter.NewReadWriter(conn)

	_, _, err = handshake.DoClient(bc, false, false)
	require.NoError(t, err)

	mrw := message.NewReadWriter(bc, bc, true)

	err = mrw.Write(&message.CommandAMF0{
		ChunkStreamID: 3,
		Name:          "connect",
		CommandID:     1,
		Arguments: []any{
			amf0.Object{
				{Key: "app", Value: "stream?key=val"},
				{Key: "flashVer", Value: "LNX 9,0,124,2"},
				{Key: "tcUrl", Value: "rtmp://127.0.0.1:9121/stream?key=val"},
				{Key: "fpad", Value: false},
				{Key: "capabilities", Value: float64(15)},
				{Key: "audioCodecs", Value: float64(4071)},
				{Key: "videoCodecs", Value: float64(252)},
				{Key: "videoFunction", Value: float64(1)},
				{Key: "fourCcList", Value: amf0.StrictArray{
					"av01",
					"Avc1",
				}},
			},
		},
	})
	require.NoError(t, err)

	<-done
}

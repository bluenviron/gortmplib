package gortmplib_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortmplib/pkg/amf0"
	"github.com/bluenviron/gortmplib/pkg/bytecounter"
	"github.com/bluenviron/gortmplib/pkg/handshake"
	"github.com/bluenviron/gortmplib/pkg/message"
)

var serverCert = []byte(`-----BEGIN CERTIFICATE-----
MIIDkzCCAnugAwIBAgIUHFnymlrkEnz3ThpFvSrqybBepn4wDQYJKoZIhvcNAQEL
BQAwWTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDESMBAGA1UEAwwJbG9jYWxob3N0MB4X
DTIxMTIwMzIxNDg0MFoXDTMxMTIwMTIxNDg0MFowWTELMAkGA1UEBhMCQVUxEzAR
BgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5
IEx0ZDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAv8h21YDIAYNzewrfQqQTlODJjuUZKxMCO7z1wIapem5I+1I8n+vD
v8qvuyZk1m9CKQPfXxhJz0TT5kECoUY0KaDtykSzfaUK34F9J1d5snDkaOtN48W+
8l39Wtcvc5JW17jNwabppAkHHYAMQryO8urKLWKbZmLhYCJdYgNqb8ciWPsnYNA0
zcnKML9zQphh7dxPq1wCsy/c/XZUzxTLAe8hsCKuqpESEX3MMJA9gOLmiOF0JgpT
9h6eqvJU8IK0QMIv3tekJWSBvTLyz4ghENs10sMKKNqR6NWt2SsOloeBkOhIDLOk
byLaPEvugrQsga99uhANRpXp+CHnVeAH8QIDAQABo1MwUTAdBgNVHQ4EFgQUwyEH
cMynEoy1/TnbIhgpEAs038gwHwYDVR0jBBgwFoAUwyEHcMynEoy1/TnbIhgpEAs0
38gwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAiV56KhDoUVzW
qV1X0QbfLaifimsN3Na3lUgmjcgyUe8rHj09pXuAD/AcQw/zwKzZ6dPtizBeNLN8
jV1dbJmR7DE3MDlndgMKTOKFsqzHjG9UTXkBGFUEM1shn2GE8XcvDF0AzKU82YjP
B0KswA1NoYTNP2PW4IhZRzv2M+fnmkvc8DSEZ+dxEMg3aJfe/WLPvYjDpFXLvuxl
YnerRQ04hFysh5eogPFpB4KyyPs6jGnQFmZCbFyk9pjKRbDPJc6FkDglkzTB6j3Q
TSfgNJswOiap13vQQKf5Vu7LTuyO4Wjfjr74QNqMLLNIgcC7n2jfQj1g5Xa0bnF5
G4tLrCLUUw==
-----END CERTIFICATE-----
`)

var serverKey = []byte(`-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC/yHbVgMgBg3N7
Ct9CpBOU4MmO5RkrEwI7vPXAhql6bkj7Ujyf68O/yq+7JmTWb0IpA99fGEnPRNPm
QQKhRjQpoO3KRLN9pQrfgX0nV3mycORo603jxb7yXf1a1y9zklbXuM3BpumkCQcd
gAxCvI7y6sotYptmYuFgIl1iA2pvxyJY+ydg0DTNycowv3NCmGHt3E+rXAKzL9z9
dlTPFMsB7yGwIq6qkRIRfcwwkD2A4uaI4XQmClP2Hp6q8lTwgrRAwi/e16QlZIG9
MvLPiCEQ2zXSwwoo2pHo1a3ZKw6Wh4GQ6EgMs6RvIto8S+6CtCyBr326EA1Glen4
IedV4AfxAgMBAAECggEAOqcJSNSA1o2oJKo3i374iiCRJAWGw/ilRzXMBtxoOow9
/7av2czV6fMH+XmNf1M5bafEiaW49Q28rH+XWVFKJK0V7DVEm5l9EMveRcjn7B3A
jSHhiVZxxlfeYwjKd1L7AjB/pMjyTXuBVJFTrplSMpKB0I2GrzJwcOExpAcdZx98
K0s5pauJH9bE0kI3p585SGQaIjrz0LvAmf6cQ5HhKfahJdWNnKZ/S4Kdqe+JCgyd
NawREHhf3tU01Cd3DOgXn4+5V/Ts6XtqY1RuSvonNv3nyeiOpX8C4cHKD5u2sNOC
3J4xWrrs0W3e8IATgAys56teKbEufHTUx52wNhAbzQKBgQD56W0tPCuaKrsjxsvE
dNHdm/9aQrN1jCJxUcGaxCIioXSyDvpSKcgxQbEqHXRTtJt5/Kadz9omq4vFTVtl
5Gf+3Lrf3ZT82SvYHtlIMdBZLlKwk6MolEa0KGAuJBNJVRIOkm5YjV/3bJebeTIb
WrLEyNCOXFAh3KVzBPU8nJ1aTwKBgQDEdISg3UsSOLBa0BfoJ5FlqGepZSufYgqh
xAJn8EbopnlzfmHBZAhE2+Igh0xcHhQqHThc3OuLtAkWu6fUSLiSA+XjU9TWPpA1
C/325rhT23fxzYIlYFegR9BToxYhv14ufkcTXRfHRAhffk7K5A2nlJfldDZRmUh2
5KIjXQ0pvwKBgQCa7S6VgFu3cw4Ym8DuxUzlCTRADGGcWYdwoLJY84YF2fmx+L8N
+ID2qDbgWOooiipocUwJQTWIC4jWg6JJhFNEGCpxZbhbF3aqwFULAHadEq6IcL4R
Bfre7LjTYeHi8C4FgpmNo/b+N/+0jmmVs6BnheZkmq3CkDqxFz3AmYai2QKBgQC1
kzAmcoJ5U/YD6YO/Khsjx3QQSBb6mCZVf5HtuVIApCVqzuvRUACojEbDY+n61j4y
8pDum64FkKA557Xl6lTVeE7ZPtlgL7EfpnbT5kmGEDobPqPEofg7h0SQmRLSnEqT
VFmjFw7sOQA4Ksjuk7vfIOMHy9KMts0YPpdxcgbBhwKBgQCP8MeRPuhZ26/oIESr
I8ArLEaPebYmLXCT2ZTudGztoyYFxinRGHA4PdamSOKfB1li52wAaqgRA3cSqkUi
kabimVOvrOAWlnvznqXEHPNx6mbbKs08jh+uRRmrOmMrxAobpTqarL2Sdxb6afID
NkxNic7oHgsZpIkZ8HK+QjAAWA==
-----END PRIVATE KEY-----
`)

func queryDecode(enc string) map[string]string {
	// do not use url.ParseQuery since values are not URL-encoded
	vals := make(map[string]string)

	for kv := range strings.SplitSeq(enc, "&") {
		tmp := strings.SplitN(kv, "=", 2)
		if len(tmp) == 2 {
			vals[tmp[0]] = tmp[1]
		}
	}

	return vals
}

func TestClientURL(t *testing.T) {
	for _, ca := range []struct {
		name              string
		url               string
		expectedTcURL     string
		expectedApp       string
		expectedStreamKey string
	}{
		{
			name:          "one component",
			url:           "rtmp://localhost:9121/app",
			expectedTcURL: "rtmp://localhost:9121/app",
			expectedApp:   "app",
		},
		{
			name:          "one component with query",
			url:           "rtmp://localhost:9121/app?query=1",
			expectedTcURL: "rtmp://localhost:9121/app?query=1",
			expectedApp:   "app?query=1",
		},
		{
			name:          "two components",
			url:           "rtmp://localhost:9121/app/stream",
			expectedTcURL: "rtmp://localhost:9121/app/stream",
			expectedApp:   "app/stream",
		},
		{
			name:          "two components with query",
			url:           "rtmp://localhost:9121/app/stream?query=1",
			expectedTcURL: "rtmp://localhost:9121/app/stream?query=1",
			expectedApp:   "app/stream?query=1",
		},
		{
			name: "fragment",
			url: "rtmp://localhost:9121/three/segment/path?delaycc=0.1" +
				"#BigBuckBunny?audioLifetime=0.25&videoLifetime=2",
			expectedTcURL:     "rtmp://localhost:9121/three/segment/path?delaycc=0.1",
			expectedApp:       "three/segment/path?delaycc=0.1",
			expectedStreamKey: "BigBuckBunny?audioLifetime=0.25&videoLifetime=2",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:9121")
			require.NoError(t, err)
			defer ln.Close()

			done := make(chan struct{})

			go func() {
				conn, err2 := ln.Accept()
				require.NoError(t, err2)
				defer conn.Close()
				bc := bytecounter.NewReadWriter(conn)

				_, _, err2 = handshake.DoServer(bc, false)
				require.NoError(t, err2)

				mrw := message.NewReadWriter(bc, bc, true)

				msg, err2 := mrw.Read()
				require.NoError(t, err2)
				require.Equal(t, &message.SetWindowAckSize{Value: 2500000}, msg)

				msg, err2 = mrw.Read()
				require.NoError(t, err2)
				require.Equal(t, &message.SetPeerBandwidth{Value: 2500000, Type: 2}, msg)

				msg, err2 = mrw.Read()
				require.NoError(t, err2)
				require.Equal(t, &message.SetChunkSize{Value: 65536}, msg)

				msg, err2 = mrw.Read()
				require.NoError(t, err2)
				connectMsg := msg.(*message.CommandAMF0)
				require.Equal(t, "connect", connectMsg.Name)
				obj := connectMsg.Arguments[0].(amf0.Object)
				actualTcURL, _ := obj.GetString("tcUrl")
				actualApp, _ := obj.GetString("app")
				require.Equal(t, ca.expectedTcURL, actualTcURL)
				require.Equal(t, ca.expectedApp, actualApp)

				err2 = mrw.Write(&message.CommandAMF0{
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
				})
				require.NoError(t, err2)

				msg, err2 = mrw.Read()
				require.NoError(t, err2)
				require.Equal(t, &message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "createStream",
					CommandID:     2,
					Arguments:     []any{nil},
				}, msg)

				err2 = mrw.Write(&message.CommandAMF0{
					ChunkStreamID: 3,
					Name:          "_result",
					CommandID:     2,
					Arguments:     []any{nil, float64(1)},
				})
				require.NoError(t, err2)

				msg, err2 = mrw.Read()
				require.NoError(t, err2)
				require.Equal(t, &message.UserControlSetBufferLength{BufferLength: 0x64}, msg)

				msg, err2 = mrw.Read()
				require.NoError(t, err2)
				playMsg := msg.(*message.CommandAMF0)
				require.Equal(t, "play", playMsg.Name)
				actualStreamKey := playMsg.Arguments[1].(string)
				require.Equal(t, ca.expectedStreamKey, actualStreamKey)

				err2 = mrw.Write(&message.CommandAMF0{
					ChunkStreamID:   5,
					MessageStreamID: 0x1000000,
					Name:            "onStatus",
					CommandID:       3,
					Arguments: []any{
						nil,
						amf0.Object{
							{Key: "level", Value: "status"},
							{Key: "code", Value: "NetStream.Play.Reset"},
							{Key: "description", Value: "play reset"},
						},
					},
				})
				require.NoError(t, err2)

				close(done)
			}()

			ur, err := url.Parse(ca.url)
			require.NoError(t, err)

			c := &gortmplib.Client{
				URL: ur,
			}
			err = c.Initialize(context.Background())
			require.NoError(t, err)
			defer c.Close()

			<-done
		})
	}
}

func TestClientReadPublish(t *testing.T) {
	for _, ca := range []string{
		"auth",
		"read",
		"read nginx rtmp",
		"read srs",
		"publish",
	} {
		t.Run(ca, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:9121")
			require.NoError(t, err)
			defer ln.Close()

			done := make(chan struct{})
			authState := 0

			go func() {
				for {
					conn, err2 := ln.Accept()
					require.NoError(t, err2)
					defer conn.Close()
					bc := bytecounter.NewReadWriter(conn)

					_, _, err2 = handshake.DoServer(bc, false)
					require.NoError(t, err2)

					mrw := message.NewReadWriter(bc, bc, true)

					msg, err2 := mrw.Read()
					require.NoError(t, err2)
					require.Equal(t, &message.SetWindowAckSize{
						Value: 2500000,
					}, msg)

					msg, err2 = mrw.Read()
					require.NoError(t, err2)
					require.Equal(t, &message.SetPeerBandwidth{
						Value: 2500000,
						Type:  2,
					}, msg)

					msg, err2 = mrw.Read()
					require.NoError(t, err2)
					require.Equal(t, &message.SetChunkSize{
						Value: 65536,
					}, msg)

					switch ca {
					case "auth":
						msg, err2 = mrw.Read()
						require.NoError(t, err2)

						switch authState {
						case 0: //nolint:dupl
							require.Equal(t, &message.CommandAMF0{ //nolint:dupl
								ChunkStreamID: 3,
								Name:          "connect",
								CommandID:     1,
								Arguments: []any{
									amf0.Object{
										{Key: "app", Value: "stream"},
										{Key: "flashVer", Value: "LNX 9,0,124,2"},
										{Key: "tcUrl", Value: "rtmp://127.0.0.1:9121/stream"},
										{Key: "objectEncoding", Value: float64(0)},
										{Key: "fpad", Value: false},
										{Key: "capabilities", Value: float64(15)},
										{Key: "audioCodecs", Value: float64(1413)},
										{Key: "videoCodecs", Value: float64(128)},
										{Key: "videoFunction", Value: float64(0)},
										{Key: "fourCcList", Value: amf0.StrictArray{
											"av01",
											"vp09",
											"hvc1",
											"avc1",
											"Opus",
											"fLaC",
											"ac-3",
											"mp4a",
											".mp3",
										}},
									},
								},
							}, msg)

						case 1: //nolint:dupl
							require.Equal(t, &message.CommandAMF0{ //nolint:dupl
								ChunkStreamID: 3,
								Name:          "connect",
								CommandID:     1,
								Arguments: []any{
									amf0.Object{
										{Key: "app", Value: "stream?authmod=adobe&user=myuser"},
										{Key: "flashVer", Value: "LNX 9,0,124,2"},
										{Key: "tcUrl", Value: "rtmp://127.0.0.1:9121/stream?authmod=adobe&user=myuser"},
										{Key: "objectEncoding", Value: float64(0)},
										{Key: "fpad", Value: false},
										{Key: "capabilities", Value: float64(15)},
										{Key: "audioCodecs", Value: float64(1413)},
										{Key: "videoCodecs", Value: float64(128)},
										{Key: "videoFunction", Value: float64(0)},
										{Key: "fourCcList", Value: amf0.StrictArray{
											"av01",
											"vp09",
											"hvc1",
											"avc1",
											"Opus",
											"fLaC",
											"ac-3",
											"mp4a",
											".mp3",
										}},
									},
								},
							}, msg)

						case 2:
							app, _ := msg.(*message.CommandAMF0).Arguments[0].(amf0.Object).GetString("app")
							query := queryDecode(app[len("stream?"):])
							clientChallenge := query["challenge"]
							response := authResponse("myuser", "mypass", "salt123", "", "server456challenge", clientChallenge)

							require.Equal(t, &message.CommandAMF0{
								ChunkStreamID: 3,
								Name:          "connect",
								CommandID:     1,
								Arguments: []any{
									amf0.Object{
										{
											Key: "app",
											Value: "stream?authmod=adobe&user=myuser&challenge=" +
												clientChallenge + "&response=" + response,
										},
										{Key: "flashVer", Value: "LNX 9,0,124,2"},
										{
											Key: "tcUrl",
											Value: "rtmp://127.0.0.1:9121/stream?authmod=adobe&user=myuser&challenge=" +
												clientChallenge + "&response=" + response,
										},
										{Key: "objectEncoding", Value: float64(0)},
										{Key: "fpad", Value: false},
										{Key: "capabilities", Value: float64(15)},
										{Key: "audioCodecs", Value: float64(1413)},
										{Key: "videoCodecs", Value: float64(128)},
										{Key: "videoFunction", Value: float64(0)},
										{Key: "fourCcList", Value: amf0.StrictArray{
											"av01",
											"vp09",
											"hvc1",
											"avc1",
											"Opus",
											"fLaC",
											"ac-3",
											"mp4a",
											".mp3",
										}},
									},
								},
							}, msg)
						}

					case "read", "read nginx rtmp", "read srs":
						msg, err2 = mrw.Read()
						require.NoError(t, err2)
						require.Equal(t, &message.CommandAMF0{ //nolint:dupl
							ChunkStreamID: 3,
							Name:          "connect",
							CommandID:     1,
							Arguments: []any{
								amf0.Object{
									{Key: "app", Value: "stream"},
									{Key: "flashVer", Value: "LNX 9,0,124,2"},
									{Key: "tcUrl", Value: "rtmp://127.0.0.1:9121/stream"},
									{Key: "objectEncoding", Value: float64(0)},
									{Key: "fpad", Value: false},
									{Key: "capabilities", Value: float64(15)},
									{Key: "audioCodecs", Value: float64(1413)},
									{Key: "videoCodecs", Value: float64(128)},
									{Key: "videoFunction", Value: float64(0)},
									{Key: "fourCcList", Value: amf0.StrictArray{
										"av01",
										"vp09",
										"hvc1",
										"avc1",
										"Opus",
										"fLaC",
										"ac-3",
										"mp4a",
										".mp3",
									}},
								},
							},
						}, msg)

					case "publish":
						msg, err2 = mrw.Read()
						require.NoError(t, err2)
						require.Equal(t, &message.CommandAMF0{
							ChunkStreamID: 3,
							Name:          "connect",
							CommandID:     1,
							Arguments: []any{
								amf0.Object{
									{Key: "app", Value: "stream"},
									{Key: "flashVer", Value: "LNX 9,0,124,2"},
									{Key: "tcUrl", Value: "rtmp://127.0.0.1:9121/stream"},
									{Key: "objectEncoding", Value: float64(0)},
								},
							},
						}, msg)
					}

					if ca == "auth" {
						switch authState {
						case 0:
							err2 = mrw.Write(&message.CommandAMF0{
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
							})
							require.NoError(t, err2)

							authState++
							continue

						case 1:
							err2 = mrw.Write(&message.CommandAMF0{
								ChunkStreamID: 3,
								Name:          "_error",
								CommandID:     1,
								Arguments: []any{
									nil,
									amf0.Object{
										{Key: "level", Value: "error"},
										{Key: "code", Value: "NetConnection.Connect.Rejected"},
										{
											Key:   "description",
											Value: "authmod=adobe ?reason=needauth&user=myuser&salt=salt123&challenge=server456challenge",
										},
									},
								},
							})
							require.NoError(t, err2)

							authState++
							continue
						}
					}

					err2 = mrw.Write(&message.CommandAMF0{
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
					})
					require.NoError(t, err2)

					switch ca {
					case "auth", "read", "read nginx rtmp", "read srs":
						msg, err2 = mrw.Read()
						require.NoError(t, err2)
						require.Equal(t, &message.CommandAMF0{
							ChunkStreamID: 3,
							Name:          "createStream",
							CommandID:     2,
							Arguments: []any{
								nil,
							},
						}, msg)

						if ca == "read srs" {
							err2 = mrw.Write(&message.CommandAMF0{
								ChunkStreamID: 3,
								Name:          "onBWDone",
								CommandID:     0,
								Arguments: []any{
									nil,
								},
							})
							require.NoError(t, err2)
						}

						err2 = mrw.Write(&message.CommandAMF0{
							ChunkStreamID: 3,
							Name:          "_result",
							CommandID:     2,
							Arguments: []any{
								nil,
								float64(1),
							},
						})
						require.NoError(t, err2)

						msg, err2 = mrw.Read()
						require.NoError(t, err2)
						require.Equal(t, &message.UserControlSetBufferLength{
							BufferLength: 0x64,
						}, msg)

						msg, err2 = mrw.Read()
						require.NoError(t, err2)
						require.Equal(t, &message.CommandAMF0{
							ChunkStreamID:   4,
							MessageStreamID: 0x1000000,
							Name:            "play",
							CommandID:       3,
							Arguments: []any{
								nil,
								"",
							},
						}, msg)

						err2 = mrw.Write(&message.CommandAMF0{
							ChunkStreamID:   5,
							MessageStreamID: 0x1000000,
							Name:            "onStatus",
							CommandID: func() int {
								if ca == "read nginx rtmp" {
									return 0
								}
								return 3
							}(),
							Arguments: []any{
								nil,
								amf0.Object{
									{Key: "level", Value: "status"},
									{Key: "code", Value: "NetStream.Play.Reset"},
									{Key: "description", Value: "play reset"},
								},
							},
						})
						require.NoError(t, err2)

					case "publish":
						msg, err2 = mrw.Read()
						require.NoError(t, err2)
						require.Equal(t, &message.CommandAMF0{
							ChunkStreamID: 3,
							Name:          "releaseStream",
							CommandID:     2,
							Arguments: []any{
								nil,
								"",
							},
						}, msg)

						msg, err2 = mrw.Read()
						require.NoError(t, err2)
						require.Equal(t, &message.CommandAMF0{
							ChunkStreamID: 3,
							Name:          "FCPublish",
							CommandID:     3,
							Arguments: []any{
								nil,
								"",
							},
						}, msg)

						msg, err2 = mrw.Read()
						require.NoError(t, err2)
						require.Equal(t, &message.CommandAMF0{
							ChunkStreamID: 3,
							Name:          "createStream",
							CommandID:     4,
							Arguments: []any{
								nil,
							},
						}, msg)

						err2 = mrw.Write(&message.CommandAMF0{
							ChunkStreamID: 3,
							Name:          "_result",
							CommandID:     4,
							Arguments: []any{
								nil,
								float64(1),
							},
						})
						require.NoError(t, err2)

						msg, err2 = mrw.Read()
						require.NoError(t, err2)
						require.Equal(t, &message.CommandAMF0{
							ChunkStreamID:   4,
							MessageStreamID: 0x1000000,
							Name:            "publish",
							CommandID:       5,
							Arguments: []any{
								nil,
								"",
								"live",
							},
						}, msg)

						err2 = mrw.Write(&message.CommandAMF0{
							ChunkStreamID:   5,
							MessageStreamID: 0x1000000,
							Name:            "onStatus",
							CommandID:       5,
							Arguments: []any{
								nil,
								amf0.Object{
									{Key: "level", Value: "status"},
									{Key: "code", Value: "NetStream.Publish.Start"},
									{Key: "description", Value: "publish start"},
								},
							},
						})
						require.NoError(t, err2)
					}

					close(done)
					break
				}
			}()

			var rawURL string

			if ca == "auth" {
				rawURL = "rtmp://myuser:mypass@127.0.0.1:9121/stream"
			} else {
				rawURL = "rtmp://127.0.0.1:9121/stream"
			}

			u, err := url.Parse(rawURL)
			require.NoError(t, err)

			c := &gortmplib.Client{
				URL:     u,
				Publish: (ca == "publish"),
			}
			err = c.Initialize(context.Background())
			require.NoError(t, err)
			defer c.Close()

			switch ca {
			case "read", "read nginx rtmp":
				require.Equal(t, uint64(3421), c.BytesReceived())
				require.Equal(t, uint64(0xdba), c.BytesSent())

			case "read srs":
				require.Equal(t, uint64(0xd7a), c.BytesReceived())
				require.Equal(t, uint64(0xdba), c.BytesSent())

			case "publish":
				require.Equal(t, uint64(3427), c.BytesReceived())
				require.Equal(t, uint64(0xd3e), c.BytesSent())
			}

			<-done
		})
	}
}

func TestClientRTMPS(t *testing.T) {
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	require.NoError(t, err)

	l, err := tls.Listen("tcp", "localhost:1936", &tls.Config{
		Certificates: []tls.Certificate{cert},
		VerifyConnection: func(cs tls.ConnectionState) error {
			// check that SNI is correctly filled by client
			require.Equal(t, "localhost", cs.ServerName)
			return nil
		},
	})
	require.NoError(t, err)
	defer l.Close()

	serverDone := make(chan struct{})
	defer func() { <-serverDone }()

	go func() {
		defer close(serverDone)

		nconn, err2 := l.Accept()
		require.NoError(t, err2)
		defer nconn.Close()
		bc := bytecounter.NewReadWriter(nconn)

		_, _, err2 = handshake.DoServer(bc, false)
		require.NoError(t, err2)

		mrw := message.NewReadWriter(bc, bc, true)

		for {
			var msg message.Message
			msg, err = mrw.Read()
			require.NoError(t, err)

			if msg, ok := msg.(*message.CommandAMF0); ok && msg.Name == "connect" {
				break
			}
		}

		err2 = mrw.Write(&message.CommandAMF0{
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
		})
		require.NoError(t, err2)

		for {
			var msg message.Message
			msg, err = mrw.Read()
			require.NoError(t, err)

			if msg, ok := msg.(*message.CommandAMF0); ok && msg.Name == "createStream" {
				break
			}
		}

		err2 = mrw.Write(&message.CommandAMF0{
			ChunkStreamID: 3,
			Name:          "_result",
			CommandID:     4,
			Arguments: []any{
				nil,
				float64(1),
			},
		})
		require.NoError(t, err2)

		for {
			var msg message.Message
			msg, err = mrw.Read()
			require.NoError(t, err)

			if msg, ok := msg.(*message.CommandAMF0); ok && msg.Name == "publish" {
				break
			}
		}

		err2 = mrw.Write(&message.CommandAMF0{
			ChunkStreamID:   5,
			MessageStreamID: 0x1000000,
			Name:            "onStatus",
			CommandID:       5,
			Arguments: []any{
				nil,
				amf0.Object{
					{Key: "level", Value: "status"},
					{Key: "code", Value: "NetStream.Publish.Start"},
					{Key: "description", Value: "publish start"},
				},
			},
		})
		require.NoError(t, err2)
	}()

	u, err := url.Parse("rtmps://localhost:1936/test")
	require.NoError(t, err)

	c := &gortmplib.Client{
		URL:     u,
		Publish: true,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	err = c.Initialize(context.Background())
	require.NoError(t, err)
	defer c.Close()
}

func TestClientDefaultPort(t *testing.T) {
	for _, ca := range []struct {
		name         string
		url          string
		expectedAddr string
	}{
		{
			name:         "rtmp",
			url:          "rtmp://localhost/app",
			expectedAddr: "localhost:1935",
		},
		{
			name:         "rtmps",
			url:          "rtmps://localhost/app",
			expectedAddr: "localhost:443",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			var dialedAddr string

			ur, err := url.Parse(ca.url)
			require.NoError(t, err)

			c := &gortmplib.Client{
				URL: ur,
				DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
					dialedAddr = address
					return nil, fmt.Errorf("dial intercepted")
				},
			}
			err = c.Initialize(context.Background())
			require.Error(t, err)
			require.Equal(t, ca.expectedAddr, dialedAddr)
		})
	}
}

package gortmplib_test

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/abema/go-mp4"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/av1"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/flac"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h265"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/opus"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortmplib/pkg/amf0"
	"github.com/bluenviron/gortmplib/pkg/bytecounter"
	"github.com/bluenviron/gortmplib/pkg/codecs"
	"github.com/bluenviron/gortmplib/pkg/message"
	"github.com/bluenviron/gortmplib/pkg/rawmessage"
)

var (
	h265DefaultVPS = []byte{
		0x40, 0x01, 0x0c, 0x01, 0xff, 0xff, 0x02, 0x20,
		0x00, 0x00, 0x03, 0x00, 0xb0, 0x00, 0x00, 0x03,
		0x00, 0x00, 0x03, 0x00, 0x7b, 0x18, 0xb0, 0x24,
	}

	h265DefaultSPS = []byte{
		0x42, 0x01, 0x01, 0x02, 0x20, 0x00, 0x00, 0x03,
		0x00, 0xb0, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
		0x00, 0x7b, 0xa0, 0x07, 0x82, 0x00, 0x88, 0x7d,
		0xb6, 0x71, 0x8b, 0x92, 0x44, 0x80, 0x53, 0x88,
		0x88, 0x92, 0xcf, 0x24, 0xa6, 0x92, 0x72, 0xc9,
		0x12, 0x49, 0x22, 0xdc, 0x91, 0xaa, 0x48, 0xfc,
		0xa2, 0x23, 0xff, 0x00, 0x01, 0x00, 0x01, 0x6a,
		0x02, 0x02, 0x02, 0x01,
	}

	h265DefaultPPS = []byte{
		0x44, 0x01, 0xc0, 0x25, 0x2f, 0x05, 0x32, 0x40,
	}

	h264DefaultSPS = []byte{ // 1920x1080 baseline
		0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02,
		0x27, 0xe5, 0x84, 0x00, 0x00, 0x03, 0x00, 0x04,
		0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9, 0x20,
	}

	h264DefaultPPS = []byte{0x08, 0x06, 0x07, 0x08}
)

func TestWriterDrainIncomingData(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	server := &dummyConn{rw: serverConn}
	server.initialize()

	client := &dummyConn{rw: clientConn}
	client.initialize()

	readDone := make(chan error, 1)
	go func() {
		_, err := client.Read()
		readDone <- err
	}()

	w := &gortmplib.Writer{Conn: server}
	err := w.Initialize()
	require.NoError(t, err)
	require.NoError(t, <-readDone)

	err = client.Write(&message.SetChunkSize{Value: 128})
	require.NoError(t, err)

	err = client.Write(&message.SetChunkSize{Value: 128})
	require.NoError(t, err)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- w.Wait()
	}()

	select {
	case err = <-waitDone:
		t.Fatalf("Wait returned before a read error: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	_, err = clientConn.Write([]byte{0})
	require.NoError(t, err)

	select {
	case err = <-waitDone:
		require.EqualError(t, err, "extended chunk stream IDs are not supported (yet)")
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return after a read error")
	}

	require.EqualError(t, w.Wait(), "extended chunk stream IDs are not supported (yet)")
}

func TestWriter(t *testing.T) {
	for _, ca := range []string{
		"h264 + aac",
		"av1",
		"vp9",
		"h265",
		"h265 no params",
		"h264 no params",
		"opus",
		"mp3",
		"ac-3",
		"flac",
		"pcma",
		"pcmu",
		"lpcm",
	} {
		t.Run(ca, func(t *testing.T) {
			var tracks []*gortmplib.Track

			switch ca {
			case "h264 + aac":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.H264{
					SPS: []byte{
						0x67, 0x64, 0x00, 0x0c, 0xac, 0x3b, 0x50, 0xb0,
						0x4b, 0x42, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00,
						0x00, 0x03, 0x00, 0x3d, 0x08,
					},
					PPS: []byte{
						0x68, 0xee, 0x3c, 0x80,
					},
				}})

				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.MPEG4Audio{
					Config: &mpeg4audio.AudioSpecificConfig{
						Type:          2,
						SampleRate:    44100,
						ChannelConfig: 2,
						ChannelCount:  2, //nolint:staticcheck
					},
				}})

			case "av1":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.AV1{}})

			case "vp9":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.VP9{}})

			case "h265":
				tracks = append(tracks, &gortmplib.Track{Codec: testCodecH265})

			case "h265 no params":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.H265{}})

			case "h264 no params":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.H264{}})

			case "opus":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.Opus{
					IDHeader: &opus.IDHeader{
						Version:      0x1,
						ChannelCount: 2,
						PreSkip:      3840,
					},
				}})

			case "mp3":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.MPEG1Audio{}})

			case "ac-3":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.AC3{
					SampleRate:   44100,
					ChannelCount: 2,
				}})

			case "flac":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.FLAC{
					StreamInfo: &flac.StreamInfo{
						MinBlockSize: 16,
						MaxBlockSize: 65535,
						SampleRate:   44100,
						ChannelCount: 2,
						BitDepth:     16,
					},
				}})

			case "pcma":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.G711{
					MULaw:        false,
					ChannelCount: 1,
				}})

			case "pcmu":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.G711{
					MULaw:        true,
					ChannelCount: 1,
				}})

			case "lpcm":
				tracks = append(tracks, &gortmplib.Track{Codec: &codecs.LPCM{
					BitDepth:     16,
					SampleRate:   44100,
					ChannelCount: 1,
				}})
			}

			var input bytes.Buffer
			var output bytes.Buffer
			c := &dummyConn{
				rw: &readWriter{
					Reader: &input,
					Writer: &output,
				},
			}
			c.initialize()

			w := &gortmplib.Writer{
				Conn:   c,
				Tracks: tracks,
			}
			err := w.Initialize()
			require.NoError(t, err)

			bc := bytecounter.NewReadWriter(&output)
			mrw := message.NewReadWriter(bc, bc, true)

			msg, err := mrw.Read()
			require.NoError(t, err)

			switch ca {
			case "h264 + aac":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(7)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(10)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "av1":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(1.635135537e+09)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(0)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "vp9":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(1.987063865e+09)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(0)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "h265":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(1.752589105e+09)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(0)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "h265 no params":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(1.752589105e+09)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(0)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "h264 no params":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(7)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(0)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "opus":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(0)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(1.332770163e+09)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "mp3":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(0)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(2)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "ac-3":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(0)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(1.633889587e+09)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "flac":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(0)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(message.FourCCFLAC)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "pcma":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(0)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(7)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "pcmu":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(0)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(8)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)

			case "lpcm":
				require.Equal(t, &message.DataAMF0{
					ChunkStreamID:   4,
					MessageStreamID: 0x1000000,
					Payload: []any{
						"@setDataFrame",
						"onMetaData",
						amf0.Object{
							{Key: "videocodecid", Value: float64(0)},
							{Key: "videodatarate", Value: float64(0)},
							{Key: "audiocodecid", Value: float64(3)},
							{Key: "audiodatarate", Value: float64(0)},
						},
					},
				}, msg)
			}

			switch ca {
			case "h264 + aac":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Video{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecH264,
					FrameType:       message.VideoFrameTypeKeyFrame,
					IsKeyFrame:      true, //nolint:staticcheck
					PacketType:      message.VideoPacketTypeConfig,
					AVCConfig: generateAvcC(t,
						[]byte{
							0x67, 0x64, 0x00, 0x0c, 0xac, 0x3b, 0x50, 0xb0,
							0x4b, 0x42, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00,
							0x00, 0x03, 0x00, 0x3d, 0x08,
						},
						[]byte{
							0x68, 0xee, 0x3c, 0x80,
						},
					),
				}, msg)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Audio{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecMPEG4Audio,
					Rate:            message.AudioRate44100,
					Depth:           message.AudioDepth16,
					IsStereo:        true,
					AACType:         message.AudioAACTypeConfig,
					AACConfig: &mpeg4audio.AudioSpecificConfig{
						Type:          2,
						SampleRate:    44100,
						ChannelConfig: 2,
						ChannelCount:  2, //nolint:staticcheck
					},
				}, msg)

				err = w.WriteH264(tracks[0], 100*time.Millisecond, 0, [][]byte{{5, 1}})
				require.NoError(t, err)

				err = w.WriteMPEG4Audio(tracks[1], 0, []byte{1, 2})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Video{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecH264,
					FrameType:       message.VideoFrameTypeKeyFrame,
					IsKeyFrame:      true, //nolint:staticcheck
					PacketType:      message.VideoPacketTypeAU,
					Type:            message.VideoTypeAU, //nolint:staticcheck
					PTSDelta:        100 * time.Millisecond,
					AU:              []byte{0, 0, 0, 2, 5, 1},
				}, msg)

			case "av1":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.VideoExSequenceStart{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FrameType:       message.VideoFrameTypeKeyFrame,
					FourCC:          message.FourCCAV1,
					AV1Config: &mp4.Av1C{
						Marker:             0x1,
						Version:            0x1,
						SeqLevelIdx0:       0x8,
						ChromaSubsamplingX: 0x1,
						ChromaSubsamplingY: 0x1,
						ConfigOBUs:         []uint8{0xa, 0xb, 0x0, 0x0, 0x0, 0x42, 0xab, 0xbf, 0xc3, 0x70, 0xb, 0xe0, 0x1},
					},
				}, msg)

				err = w.WriteAV1(tracks[0], 0, [][]byte{{1, 2}})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.VideoExFramesX{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FrameType:       message.VideoFrameTypeInterFrame,
					FourCC:          message.FourCCAV1,
					Payload:         []byte{0x12, 0x0, 0x3, 0x1, 0x2},
				}, msg)

			case "vp9":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.VideoExSequenceStart{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FrameType:       message.VideoFrameTypeKeyFrame,
					FourCC:          message.FourCCVP9,
					VP9Config: &mp4.VpcC{
						FullBox:                 mp4.FullBox{Version: 0x1},
						Level:                   0x28,
						BitDepth:                0x8,
						ChromaSubsampling:       0x1,
						ColourPrimaries:         0x2,
						TransferCharacteristics: 0x2,
						MatrixCoefficients:      0x2,
						CodecInitializationData: []uint8{},
					},
				}, msg)

				err = w.WriteVP9(tracks[0], 0, []byte{1, 2})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.VideoExFramesX{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FrameType:       message.VideoFrameTypeInterFrame,
					FourCC:          message.FourCCVP9,
					Payload:         []byte{0x1, 0x2},
				}, msg)

			case "h265":
				msg, err = mrw.Read()
				require.NoError(t, err)

				require.Equal(t, &message.VideoExSequenceStart{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FrameType:       message.VideoFrameTypeKeyFrame,
					FourCC:          message.FourCCHEVC,
					HEVCConfig: &mp4.HvcC{
						ConfigurationVersion: 0x1,
						GeneralProfileIdc:    2,
						GeneralProfileCompatibility: [32]bool{
							false, false, true, false, false, false, false, false,
							false, false, false, false, false, false, false, false,
							false, false, false, false, false, false, false, false,
							false, false, false, false, false, false, false, false,
						},
						GeneralConstraintIndicator: [6]uint8{0xb0, 0x0, 0x0, 0x0, 0x0, 0x0},
						GeneralLevelIdc:            0x7b,
						Reserved1:                  0b1111,
						Reserved2:                  0b111111,
						Reserved3:                  0b111111,
						ChromaFormatIdc:            0x1,
						Reserved4:                  0b11111,
						BitDepthLumaMinus8:         2,
						Reserved5:                  0b11111,
						BitDepthChromaMinus8:       2,
						TemporalIdNested:           1,
						LengthSizeMinusOne:         0x3,
						NumTemporalLayers:          1,
						NumOfNaluArrays:            0x3,
						NaluArrays: []mp4.HEVCNaluArray{
							{
								NaluType: 0x20,
								NumNalus: 0x1,
								Nalus: []mp4.HEVCNalu{{
									Length:  24,
									NALUnit: testCodecH265.VPS,
								}},
							},
							{
								NaluType: 0x21,
								NumNalus: 0x1,
								Nalus: []mp4.HEVCNalu{{
									Length:  60,
									NALUnit: testCodecH265.SPS,
								}},
							},
							{
								NaluType: 0x22,
								NumNalus: 0x1,
								Nalus: []mp4.HEVCNalu{{
									Length:  8,
									NALUnit: testCodecH265.PPS,
								}},
							},
						},
					},
				}, msg)

				err = w.WriteH265(tracks[0], 0, 0, [][]byte{{1, 2}})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.VideoExFramesX{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FrameType:       message.VideoFrameTypeInterFrame,
					FourCC:          message.FourCCHEVC,
					Payload:         []byte{0, 0, 0, 2, 1, 2},
				}, msg)

			case "h265 no params":
				msg, err = mrw.Read()
				require.NoError(t, err)

				require.Equal(t, &message.VideoExSequenceStart{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FrameType:       message.VideoFrameTypeKeyFrame,
					FourCC:          message.FourCCHEVC,
					HEVCConfig: generateHvcC(t, h265DefaultVPS,
						h265DefaultSPS, h265DefaultPPS),
				}, msg)

				err = w.WriteH265(tracks[0], 0, 0, [][]byte{{1, 2}})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.VideoExFramesX{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FrameType:       message.VideoFrameTypeInterFrame,
					FourCC:          message.FourCCHEVC,
					Payload:         []byte{0, 0, 0, 2, 1, 2},
				}, msg)

			case "h264 no params":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Video{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecH264,
					FrameType:       message.VideoFrameTypeKeyFrame,
					IsKeyFrame:      true, //nolint:staticcheck
					PacketType:      message.VideoPacketTypeConfig,
					AVCConfig:       generateAvcC(t, h264DefaultSPS, h264DefaultPPS),
				}, msg)

				err = w.WriteH264(tracks[0], 0, 0, [][]byte{{1, 2}})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Video{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecH264,
					FrameType:       message.VideoFrameTypeInterFrame,
					PacketType:      message.VideoPacketTypeAU,
					Type:            message.VideoTypeAU, //nolint:staticcheck
					AU:              []byte{0, 0, 0, 2, 1, 2},
				}, msg)

			case "opus":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.AudioExSequenceStart{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCOpus,
					OpusConfig: &opus.IDHeader{
						Version:             1,
						PreSkip:             3840,
						ChannelCount:        2,
						ChannelMappingTable: []uint8{},
					},
				}, msg)

				err = w.WriteOpus(tracks[0], 0, []byte{1, 2})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.AudioExCodedFrames{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCOpus,
					Payload:         []byte{1, 2},
				}, msg)

			case "mp3":
				fr := []byte{
					0xff, 0xfa, 0x52, 0x04, 0x00,
				}

				err = w.WriteMPEG1Audio(tracks[0], 0, fr)
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Audio{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecMPEG1Audio,
					Rate:            message.AudioRate44100,
					Depth:           message.AudioDepth16,
					IsStereo:        true,
					AU: []byte{
						0xff, 0xfa, 0x52, 0x04, 0x00,
					},
				}, msg)

			case "ac-3":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.AudioExSequenceStart{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCAC3,
				}, msg)

				err = w.WriteAC3(tracks[0], 0, []byte{1, 2})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.AudioExCodedFrames{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCAC3,
					Payload:         []byte{1, 2},
				}, msg)

			case "flac":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.AudioExSequenceStart{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCFLAC,
					FlacConfig: &flac.StreamInfo{
						MinBlockSize: 16,
						MaxBlockSize: 65535,
						SampleRate:   44100,
						ChannelCount: 2,
						BitDepth:     16,
					},
				}, msg)

				err = w.WriteFLAC(tracks[0], 0, []byte{1, 2})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.AudioExCodedFrames{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCFLAC,
					Payload:         []byte{1, 2},
				}, msg)

			case "pcma":
				err = w.WriteG711(tracks[0], 0, []byte{1, 2})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Audio{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecPCMA,
					Depth:           message.AudioDepth16,
					AU:              []byte{1, 2},
				}, msg)

			case "pcmu":
				err = w.WriteG711(tracks[0], 0, []byte{1, 2})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Audio{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecPCMU,
					Depth:           message.AudioDepth16,
					AU:              []byte{1, 2},
				}, msg)

			case "lpcm":
				err = w.WriteLPCM(tracks[0], 0, []byte{1, 2})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Audio{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecLPCM,
					Rate:            message.AudioRate44100,
					Depth:           message.AudioDepth16,
					AU:              []byte{2, 1},
				}, msg)
			}
		})
	}
}

type readWriter struct {
	io.Reader
	io.Writer
}

// writeAndReadVideo initializes a Writer with the given tracks, calls write
// and returns the bodies of the video messages that have been sent.
func writeAndReadVideo(
	t *testing.T,
	tracks []*gortmplib.Track,
	write func(w *gortmplib.Writer) error,
) [][]byte {
	t.Helper()

	var input bytes.Buffer
	var output bytes.Buffer
	c := &dummyConn{
		rw: &readWriter{
			Reader: &input,
			Writer: &output,
		},
	}
	c.initialize()

	w := &gortmplib.Writer{
		Conn:   c,
		Tracks: tracks,
	}
	err := w.Initialize()
	require.NoError(t, err)

	err = write(w)
	require.NoError(t, err)

	bc := bytecounter.NewReader(&output)
	r := &rawmessage.Reader{
		BR:  bc,
		BCR: bc,
	}
	r.Initialize()

	var bodies [][]byte

	for {
		var msg *rawmessage.Message
		msg, err = r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		if msg.Type == uint8(message.TypeVideo) {
			bodies = append(bodies, msg.Body)
		}
	}

	return bodies
}

func TestWriterVideoFrameType(t *testing.T) {
	h265IDR := []byte{byte(h265.NALUType_IDR_W_RADL) << 1, 0x01}
	h265NonIDR := []byte{byte(h265.NALUType_TRAIL_R) << 1, 0x01}
	h264IDR := []byte{byte(h264.NALUTypeIDR), 0x01}
	h264NonIDR := []byte{byte(h264.NALUTypeNonIDR), 0x01}
	av1SequenceHeader := []byte{byte(av1.OBUTypeSequenceHeader) << 3, 0x01}
	av1Frame := []byte{6 << 3, 0x01} // OBU_FRAME
	vp9KeyFrame := []byte{
		0x82, 0x49, 0x83, 0x42, 0x00, 0x77, 0xf0, 0x32,
		0x34, 0x30, 0x38, 0x24, 0x1c, 0x19, 0x40, 0x18,
		0x03, 0x40, 0x5f, 0xb4,
	}
	vp9InterFrame := []byte{0x86, 0x00}

	for _, ca := range []struct {
		name   string
		tracks []*gortmplib.Track
		write  func(w *gortmplib.Writer, tracks []*gortmplib.Track) error
		// expected prefix of each video message body.
		// 0x90 = isExVideoHeader | KeyFrame | SequenceStart
		// 0x93 = isExVideoHeader | KeyFrame | CodedFramesX
		// 0xa3 = isExVideoHeader | InterFrame | CodedFramesX
		headers [][]byte
	}{
		{
			"h265",
			[]*gortmplib.Track{{Codec: testCodecH265}},
			func(w *gortmplib.Writer, tracks []*gortmplib.Track) error {
				err := w.WriteH265(tracks[0], 0, 0, [][]byte{h265IDR})
				if err != nil {
					return err
				}
				return w.WriteH265(tracks[0], 33*time.Millisecond, 33*time.Millisecond, [][]byte{h265NonIDR})
			},
			[][]byte{{0x90}, {0x93}, {0xa3}},
		},
		{
			"h265 with pts delta",
			[]*gortmplib.Track{{Codec: testCodecH265}},
			func(w *gortmplib.Writer, tracks []*gortmplib.Track) error {
				err := w.WriteH265(tracks[0], 66*time.Millisecond, 0, [][]byte{h265IDR})
				if err != nil {
					return err
				}
				return w.WriteH265(tracks[0], 100*time.Millisecond, 33*time.Millisecond, [][]byte{h265NonIDR})
			},
			[][]byte{{0x90}, {0x91}, {0xa1}},
		},
		{
			"av1",
			[]*gortmplib.Track{{Codec: &codecs.AV1{}}},
			func(w *gortmplib.Writer, tracks []*gortmplib.Track) error {
				err := w.WriteAV1(tracks[0], 0, [][]byte{av1SequenceHeader, av1Frame})
				if err != nil {
					return err
				}
				return w.WriteAV1(tracks[0], 33*time.Millisecond, [][]byte{av1Frame})
			},
			[][]byte{{0x90}, {0x93}, {0xa3}},
		},
		{
			"vp9",
			[]*gortmplib.Track{{Codec: &codecs.VP9{}}},
			func(w *gortmplib.Writer, tracks []*gortmplib.Track) error {
				err := w.WriteVP9(tracks[0], 0, vp9KeyFrame)
				if err != nil {
					return err
				}
				return w.WriteVP9(tracks[0], 33*time.Millisecond, vp9InterFrame)
			},
			[][]byte{{0x90}, {0x93}, {0xa3}},
		},
		{
			"h264 secondary track",
			[]*gortmplib.Track{{Codec: testCodecH264}, {Codec: testCodecH264}},
			func(w *gortmplib.Writer, tracks []*gortmplib.Track) error {
				err := w.WriteH264(tracks[1], 0, 0, [][]byte{h264IDR})
				if err != nil {
					return err
				}
				return w.WriteH264(tracks[1], 33*time.Millisecond, 33*time.Millisecond, [][]byte{h264NonIDR})
			},
			[][]byte{
				{0x17},       // legacy header of the first track: KeyFrame | AVC
				{0x96, 0x00}, // Multitrack, KeyFrame; OneTrack, SequenceStart
				{0x96, 0x03}, // Multitrack, KeyFrame; OneTrack, CodedFramesX
				{0xa6, 0x03}, // Multitrack, InterFrame; OneTrack, CodedFramesX
			},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			bodies := writeAndReadVideo(t, ca.tracks, func(w *gortmplib.Writer) error {
				return ca.write(w, ca.tracks)
			})

			require.Len(t, bodies, len(ca.headers))
			for i, header := range ca.headers {
				require.Equal(t, header, bodies[i][:len(header)])
			}
		})
	}
}

func TestWriterH265ConfigFFmpeg(t *testing.T) {
	// The expected hvcC is the one written by FFmpeg 7.1 for the same VPS/SPS/PPS,
	// that is the body of the video SequenceStart tag of out.flv after the ExVideoTagHeader.
	//
	// main:
	//   ffmpeg -f lavfi -i testsrc2=size=1280x720:rate=30 -frames:v 30 \
	//     -c:v libx265 -preset ultrafast -x265-params keyint=30:bframes=0 -f hevc in.h265
	// main 10 high tier:
	//   ffmpeg -f lavfi -i testsrc2=size=1920x1080:rate=30 -frames:v 30 -pix_fmt yuv420p10le -c:v libx265 \
	//     -preset ultrafast -x265-params keyint=30:bframes=0:level-idc=4:high-tier=1 -f hevc in.h265
	// then:
	//   ffmpeg -i in.h265 -c copy -f flv out.flv
	for _, ca := range []struct {
		name string
		vps  []byte
		sps  []byte
		pps  []byte
		hvcC []byte
	}{
		{
			"main",
			[]byte{
				0x40, 0x01, 0x0c, 0x01, 0xff, 0xff, 0x01, 0x60,
				0x00, 0x00, 0x03, 0x00, 0x90, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x03, 0x00, 0x5d, 0xba, 0x02, 0x40,
			},
			[]byte{
				0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x03,
				0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
				0x00, 0x5d, 0xa0, 0x02, 0x80, 0x80, 0x2d, 0x16,
				0x5b, 0xa4, 0xa4, 0xc2, 0xf0, 0x16, 0x80, 0x80,
				0x00, 0x00, 0x03, 0x00, 0x80, 0x00, 0x00, 0x0f,
				0x04,
			},
			[]byte{
				0x44, 0x01, 0xc0, 0x73, 0xc1, 0x89,
			},
			[]byte{
				0x01, 0x01, 0x60, 0x00, 0x00, 0x00, 0x90, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x5d, 0xf0, 0x00, 0xfc,
				0xfd, 0xf8, 0xf8, 0x00, 0x00, 0x0f, 0x03, 0x20,
				0x00, 0x01, 0x00, 0x18, 0x40, 0x01, 0x0c, 0x01,
				0xff, 0xff, 0x01, 0x60, 0x00, 0x00, 0x03, 0x00,
				0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
				0x5d, 0xba, 0x02, 0x40, 0x21, 0x00, 0x01, 0x00,
				0x29, 0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00,
				0x03, 0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00,
				0x03, 0x00, 0x5d, 0xa0, 0x02, 0x80, 0x80, 0x2d,
				0x16, 0x5b, 0xa4, 0xa4, 0xc2, 0xf0, 0x16, 0x80,
				0x80, 0x00, 0x00, 0x03, 0x00, 0x80, 0x00, 0x00,
				0x0f, 0x04, 0x22, 0x00, 0x01, 0x00, 0x06, 0x44,
				0x01, 0xc0, 0x73, 0xc1, 0x89,
			},
		},
		{
			"main 10 high tier",
			[]byte{
				0x40, 0x01, 0x0c, 0x01, 0xff, 0xff, 0x22, 0x20,
				0x00, 0x00, 0x03, 0x00, 0x90, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x03, 0x00, 0x78, 0xba, 0x02, 0x40,
			},
			[]byte{
				0x42, 0x01, 0x01, 0x22, 0x20, 0x00, 0x00, 0x03,
				0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
				0x00, 0x78, 0xa0, 0x03, 0xc0, 0x80, 0x11, 0x07,
				0xca, 0xd9, 0x6e, 0x92, 0x93, 0x0b, 0xc0, 0x5a,
				0x02, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00, 0x00,
				0x03, 0x00, 0x3c, 0x10,
			},
			[]byte{
				0x44, 0x01, 0xc0, 0x73, 0xc1, 0x89,
			},
			[]byte{
				0x01, 0x22, 0x20, 0x00, 0x00, 0x00, 0x90, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x78, 0xf0, 0x00, 0xfc,
				0xfd, 0xfa, 0xfa, 0x00, 0x00, 0x0f, 0x03, 0x20,
				0x00, 0x01, 0x00, 0x18, 0x40, 0x01, 0x0c, 0x01,
				0xff, 0xff, 0x22, 0x20, 0x00, 0x00, 0x03, 0x00,
				0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
				0x78, 0xba, 0x02, 0x40, 0x21, 0x00, 0x01, 0x00,
				0x2c, 0x42, 0x01, 0x01, 0x22, 0x20, 0x00, 0x00,
				0x03, 0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00,
				0x03, 0x00, 0x78, 0xa0, 0x03, 0xc0, 0x80, 0x11,
				0x07, 0xca, 0xd9, 0x6e, 0x92, 0x93, 0x0b, 0xc0,
				0x5a, 0x02, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00,
				0x00, 0x03, 0x00, 0x3c, 0x10, 0x22, 0x00, 0x01,
				0x00, 0x06, 0x44, 0x01, 0xc0, 0x73, 0xc1, 0x89,
			},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			tracks := []*gortmplib.Track{{Codec: &codecs.H265{
				VPS: ca.vps,
				SPS: ca.sps,
				PPS: ca.pps,
			}}}

			bodies := writeAndReadVideo(t, tracks, func(*gortmplib.Writer) error { return nil })

			require.Len(t, bodies, 1)
			require.Equal(t, []byte{'h', 'v', 'c', '1'}, bodies[0][1:5])
			require.Equal(t, ca.hvcC, bodies[0][5:])
		})
	}
}

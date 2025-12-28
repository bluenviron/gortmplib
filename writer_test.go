package gortmplib

import (
	"bytes"
	"testing"
	"time"

	"github.com/abema/go-mp4"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/gortmplib/pkg/amf0"
	"github.com/bluenviron/gortmplib/pkg/bytecounter"
	"github.com/bluenviron/gortmplib/pkg/codecs"
	"github.com/bluenviron/gortmplib/pkg/message"
)

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
		"pcma",
		"pcmu",
		"lpcm",
	} {
		t.Run(ca, func(t *testing.T) {
			var tracks []*Track

			switch ca {
			case "h264 + aac":
				tracks = append(tracks, &Track{Codec: &codecs.H264{
					SPS: []byte{
						0x67, 0x64, 0x00, 0x0c, 0xac, 0x3b, 0x50, 0xb0,
						0x4b, 0x42, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00,
						0x00, 0x03, 0x00, 0x3d, 0x08,
					},
					PPS: []byte{
						0x68, 0xee, 0x3c, 0x80,
					},
				}})

				tracks = append(tracks, &Track{Codec: &codecs.MPEG4Audio{
					Config: &mpeg4audio.AudioSpecificConfig{
						Type:         2,
						SampleRate:   44100,
						ChannelCount: 2,
					},
				}})

			case "av1":
				tracks = append(tracks, &Track{Codec: &codecs.AV1{}})

			case "vp9":
				tracks = append(tracks, &Track{Codec: &codecs.VP9{}})

			case "h265":
				tracks = append(tracks, &Track{Codec: testCodecH265})

			case "h265 no params":
				tracks = append(tracks, &Track{Codec: &codecs.H265{}})

			case "h264 no params":
				tracks = append(tracks, &Track{Codec: &codecs.H264{}})

			case "opus":
				tracks = append(tracks, &Track{Codec: &codecs.Opus{
					ChannelCount: 2,
				}})

			case "mp3":
				tracks = append(tracks, &Track{Codec: &codecs.MPEG1Audio{}})

			case "ac-3":
				tracks = append(tracks, &Track{Codec: &codecs.AC3{
					SampleRate:   44100,
					ChannelCount: 2,
				}})

			case "pcma":
				tracks = append(tracks, &Track{Codec: &codecs.G711{
					MULaw:        false,
					ChannelCount: 1,
				}})

			case "pcmu":
				tracks = append(tracks, &Track{Codec: &codecs.G711{
					MULaw:        true,
					ChannelCount: 1,
				}})

			case "lpcm":
				tracks = append(tracks, &Track{Codec: &codecs.LPCM{
					BitDepth:     16,
					SampleRate:   44100,
					ChannelCount: 1,
				}})
			}

			var buf bytes.Buffer
			c := &dummyConn{
				rw: &buf,
			}
			c.initialize()

			w := &Writer{
				Conn:   c,
				Tracks: tracks,
			}
			err := w.Initialize()
			require.NoError(t, err)

			bc := bytecounter.NewReadWriter(&buf)
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
					IsKeyFrame:      true,
					Type:            message.VideoTypeConfig,
					Payload: []byte{
						0x01, 0x64, 0x00, 0x0c, 0xff, 0xe1, 0x00, 0x15,
						0x67, 0x64, 0x00, 0x0c, 0xac, 0x3b, 0x50, 0xb0,
						0x4b, 0x42, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00,
						0x00, 0x03, 0x00, 0x3d, 0x08, 0x01, 0x00, 0x04,
						0x68, 0xee, 0x3c, 0x80,
					},
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
					Payload:         []byte{0x12, 0x10},
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
					IsKeyFrame:      true,
					Type:            message.VideoTypeAU,
					PTSDelta:        100 * time.Millisecond,
					Payload:         []byte{0, 0, 0, 2, 5, 1},
				}, msg)

			case "av1":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.VideoExSequenceStart{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCAV1,
					AV1Header: &mp4.Av1C{
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
					FourCC:          message.FourCCAV1,
					Payload:         []byte{0x12, 0x0, 0x3, 0x1, 0x2},
				}, msg)

			case "vp9":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.VideoExSequenceStart{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCVP9,
					VP9Header: &mp4.VpcC{
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
					FourCC:          message.FourCCVP9,
					Payload:         []byte{0x1, 0x2},
				}, msg)

			case "h265":
				msg, err = mrw.Read()
				require.NoError(t, err)

				require.Equal(t, &message.VideoExSequenceStart{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCHEVC,
					HEVCHeader: &mp4.HvcC{
						ConfigurationVersion: 0x1,
						GeneralProfileIdc:    2,
						GeneralProfileCompatibility: [32]bool{
							false, false, true, false, false, false, false, false,
							false, false, false, false, false, false, false, false,
							false, false, false, false, false, false, false, false,
							false, false, false, false, false, false, false, false,
						},
						GeneralConstraintIndicator: [6]uint8{0x03, 0x0, 0xb0, 0x0, 0x0, 0x03},
						GeneralLevelIdc:            0x7b,
						ChromaFormatIdc:            0x1,
						LengthSizeMinusOne:         0x3,
						NumOfNaluArrays:            0x3,
						BitDepthLumaMinus8:         2,
						BitDepthChromaMinus8:       2,
						NumTemporalLayers:          1,
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
					FourCC:          message.FourCCHEVC,
					Payload:         []byte{0, 0, 0, 2, 1, 2},
				}, msg)

			case "h265 no params":
				msg, err = mrw.Read()
				require.NoError(t, err)

				require.Equal(t, &message.VideoExSequenceStart{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCHEVC,
					HEVCHeader: generateHvcC(h265DefaultVPS,
						h265DefaultSPS, h265DefaultPPS),
				}, msg)

				err = w.WriteH265(tracks[0], 0, 0, [][]byte{{1, 2}})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.VideoExFramesX{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
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
					IsKeyFrame:      true,
					Type:            message.VideoTypeConfig,
					Payload: []byte{
						0x01, 0x42, 0xc0, 0x28, 0xff, 0xe1, 0x00, 0x19,
						0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02,
						0x27, 0xe5, 0x84, 0x00, 0x00, 0x03, 0x00, 0x04,
						0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9,
						0x20, 0x01, 0x00, 0x04, 0x08, 0x06, 0x07, 0x08,
					},
				}, msg)

				err = w.WriteH264(tracks[0], 0, 0, [][]byte{{1, 2}})
				require.NoError(t, err)

				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.Video{
					ChunkStreamID:   message.VideoChunkStreamID,
					MessageStreamID: 0x1000000,
					Codec:           message.CodecH264,
					Type:            message.VideoTypeAU,
					Payload:         []byte{0, 0, 0, 2, 1, 2},
				}, msg)

			case "opus":
				msg, err = mrw.Read()
				require.NoError(t, err)
				require.Equal(t, &message.AudioExSequenceStart{
					ChunkStreamID:   message.AudioChunkStreamID,
					MessageStreamID: 0x1000000,
					FourCC:          message.FourCCOpus,
					OpusHeader: &message.OpusIDHeader{
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
					Payload: []byte{
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
					Payload:         []byte{1, 2},
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
					Payload:         []byte{1, 2},
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
					Payload:         []byte{2, 1},
				}, msg)
			}
		})
	}
}

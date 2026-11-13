package handshake_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bluenviron/gortmplib/pkg/handshake"
)

type testReadWriter struct {
	ch chan []byte
}

func (rw *testReadWriter) Read(p []byte) (int, error) {
	in := <-rw.ch
	n := copy(p, in)
	return n, nil
}

func (rw *testReadWriter) Write(p []byte) (int, error) {
	rw.ch <- p
	return len(p), nil
}

func TestHandshake(t *testing.T) {
	for _, ca := range []string{"plain", "encrypted"} {
		t.Run(ca, func(t *testing.T) {
			rw := &testReadWriter{ch: make(chan []byte)}
			var serverInKey []byte
			var serverOutKey []byte
			done := make(chan struct{})

			go func() {
				var err error
				serverInKey, serverOutKey, err = handshake.DoServer(rw, true)
				require.NoError(t, err)
				close(done)
			}()

			clientInKey, clientOutKey, err := handshake.DoClient(rw, ca == "encrypted", true)
			require.NoError(t, err)

			<-done

			if ca == "encrypted" {
				require.NotNil(t, serverInKey)
				require.Equal(t, serverInKey, clientOutKey)
				require.Equal(t, serverOutKey, clientInKey)
			}
		})
	}
}

type splitReadWriter struct {
	reader bytes.Reader
	writer bytes.Buffer
}

func newSplitReadWriter(input []byte) *splitReadWriter {
	return &splitReadWriter{reader: *bytes.NewReader(input)}
}

func (rw *splitReadWriter) Read(p []byte) (int, error) {
	return rw.reader.Read(p)
}

func (rw *splitReadWriter) Write(p []byte) (int, error) {
	return rw.writer.Write(p)
}

func testCryptoRandRead(p []byte) (int, error) {
	for i := range p {
		p[i] = 0x01
	}
	return len(p), nil
}

func FuzzDoClient(f *testing.F) {
	f.Fuzz(func(t *testing.T, encrypted bool, strict bool, input []byte) {
		rw := newSplitReadWriter(input)
		inKey, outKey, err := handshake.DoClientWithRand(rw, encrypted, strict, testCryptoRandRead)
		if err != nil {
			return
		}

		if encrypted {
			require.Len(t, inKey, 16)
			require.Len(t, outKey, 16)
		} else {
			require.Nil(t, inKey)
			require.Nil(t, outKey)
		}
	})
}

func FuzzDoServer(f *testing.F) {
	f.Fuzz(func(t *testing.T, strict bool, input []byte) {
		rw := newSplitReadWriter(input)
		inKey, outKey, err := handshake.DoServerWithRand(rw, strict, testCryptoRandRead)
		if err != nil {
			return
		}

		if len(input) > 0 && input[0] == 6 {
			require.Len(t, inKey, 16)
			require.Len(t, outKey, 16)
		} else {
			require.Nil(t, inKey)
			require.Nil(t, outKey)
		}
	})
}

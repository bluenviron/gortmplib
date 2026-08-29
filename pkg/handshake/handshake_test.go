package handshake

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
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
				serverInKey, serverOutKey, err = DoServer(rw, true)
				require.NoError(t, err)
				close(done)
			}()

			clientInKey, clientOutKey, err := DoClient(rw, ca == "encrypted", true)
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

func setCryptoRandRead(t testing.TB) {
	t.Helper()

	previous := cryptoRandRead
	cryptoRandRead = testCryptoRandRead
	t.Cleanup(func() {
		cryptoRandRead = previous
	})
}

func FuzzDoClient(f *testing.F) {
	setCryptoRandRead(f)

	f.Fuzz(func(t *testing.T, encrypted bool, strict bool, input []byte) {
		rw := newSplitReadWriter(input)
		inKey, outKey, err := DoClient(rw, encrypted, strict)
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
	setCryptoRandRead(f)

	f.Fuzz(func(t *testing.T, strict bool, input []byte) {
		rw := newSplitReadWriter(input)
		inKey, outKey, err := DoServer(rw, strict)
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

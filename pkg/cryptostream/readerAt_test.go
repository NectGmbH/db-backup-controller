package cryptostream

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderAt(t *testing.T) {
	encpass := []byte("password")
	salt, err := getRandomSalt()
	require.NoError(t, err)

	key, iv, err := deriveKeyIV(encpass, salt)
	require.NoError(t, err)

	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	ctr := cipher.NewCTR(block, iv)

	rawData := make([]byte, 50*aes.BlockSize+6)
	_, err = rand.Read(rawData)
	require.NoError(t, err)

	encData := make([]byte, 50*aes.BlockSize+6)
	ctr.XORKeyStream(encData, rawData)

	testData := bytes.Join([][]byte{
		header,
		salt,
		encData,
	}, []byte{})

	// Let the tests begin

	t.Run("read first block", func(t *testing.T) {
		r, err := NewReaderAt(bytes.NewReader(testData), encpass)
		require.NoError(t, err)

		vData, err := io.ReadAll(io.NewSectionReader(r, 0, aes.BlockSize))
		require.NoError(t, err)

		assert.Equal(t, rawData[:aes.BlockSize], vData)
	})

	t.Run("read third block", func(t *testing.T) {
		r, err := NewReaderAt(bytes.NewReader(testData), encpass)
		require.NoError(t, err)

		vData, err := io.ReadAll(io.NewSectionReader(r, 2*aes.BlockSize, aes.BlockSize))
		require.NoError(t, err)

		assert.Equal(t, rawData[2*aes.BlockSize:3*aes.BlockSize], vData)
	})

	t.Run("read everything", func(t *testing.T) {
		r, err := NewReaderAt(bytes.NewReader(testData), encpass)
		require.NoError(t, err)

		vData, err := io.ReadAll(io.NewSectionReader(r, 0, int64(len(rawData))))
		require.NoError(t, err)

		assert.Equal(t, rawData, vData)
	})

	t.Run("read directly into slice", func(t *testing.T) {
		r, err := NewReaderAt(bytes.NewReader(testData), encpass)
		require.NoError(t, err)

		vData := make([]byte, 95)
		n, err := r.ReadAt(vData, 34)
		require.NoError(t, err)
		assert.Equal(t, 94, n)

		assert.Equal(t, rawData[34:128], vData[0:94])
	})

	t.Run("read directly into huge slice", func(t *testing.T) {
		r, err := NewReaderAt(bytes.NewReader(testData), encpass)
		require.NoError(t, err)

		vData := make([]byte, 8192)
		n, err := r.ReadAt(vData, 0)
		require.NoError(t, err)
		assert.Equal(t, len(rawData), n)

		assert.Equal(t, rawData, vData[:n])
	})
}

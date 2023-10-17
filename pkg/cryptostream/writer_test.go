package cryptostream

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	encpass := []byte("password")

	t.Run("exactly one block", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w, err := NewWriter(buf, encpass)
		require.NoError(t, err)

		assert.Equal(t, aes.BlockSize, buf.Len())

		n, err := w.Write([]byte("0123456789012345")) // 16 bytes = aes.BlockSize
		require.NoError(t, err)
		assert.Equal(t, 16, n)

		require.NoError(t, w.Close())
		assert.Equal(t, 2*aes.BlockSize, buf.Len())

		assert.Equal(t, header, buf.Bytes()[0:len(header)])
	})

	t.Run("half a block", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w, err := NewWriter(buf, encpass)
		require.NoError(t, err)

		assert.Equal(t, aes.BlockSize, buf.Len())

		n, err := w.Write([]byte("01234567")) // 8 bytes = 0.5 * aes.BlockSize
		require.NoError(t, err)
		assert.Equal(t, 8, n)

		assert.Equal(t, 16, buf.Len()) // Incomplete blocks are written on close

		require.NoError(t, w.Close())
		assert.Equal(t, 24, buf.Len())
	})

	t.Run("many blocks plus extra data", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w, err := NewWriter(buf, encpass)
		require.NoError(t, err)

		assert.Equal(t, aes.BlockSize, buf.Len())

		data := make([]byte, 50*aes.BlockSize+6)
		dn, err := rand.Read(data)
		require.NoError(t, err)

		n, err := w.Write(data)
		require.NoError(t, err)
		assert.Equal(t, dn, n)

		assert.Equal(t, 51*aes.BlockSize, buf.Len()) // Incomplete blocks are written on close

		require.NoError(t, w.Close())
		assert.Equal(t, 51*aes.BlockSize+6, buf.Len())
	})

	t.Run("many incomplete blocks", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w, err := NewWriter(buf, encpass)
		require.NoError(t, err)

		assert.Equal(t, aes.BlockSize, buf.Len())

		data := make([]byte, 50*aes.BlockSize+6)
		_, err = rand.Read(data)
		require.NoError(t, err)

		var pos int
		for _, i := range []int{15, 22, 1, 7, 127, 3, 84, 16, 289, 13, 23, 189, 17} {
			n, err := w.Write(data[pos : pos+i])
			require.NoError(t, err)
			assert.Equal(t, i, n)
			pos += i
		}

		assert.Equal(t, 51*aes.BlockSize, buf.Len()) // Incomplete blocks are written on close

		require.NoError(t, w.Close())
		assert.Equal(t, 51*aes.BlockSize+6, buf.Len())
	})
}

func TestWriterStdlibCTRCompat(t *testing.T) {
	buf := new(bytes.Buffer)
	encpass := []byte("password")

	// Create test env, no asserts, those are covered in other tests
	w, err := NewWriter(buf, encpass)
	require.NoError(t, err)

	data := make([]byte, 50*aes.BlockSize+6)
	_, err = rand.Read(data)
	require.NoError(t, err)

	_, err = w.Write(data)
	require.NoError(t, err)

	require.NoError(t, w.Close())

	// Now we check the crpyto against stdlib
	salt := buf.Bytes()[8:16]
	key, iv, err := deriveKeyIV(encpass, salt)
	require.NoError(t, err)

	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	ctr := cipher.NewCTR(block, iv)

	vData := buf.Bytes()[aes.BlockSize:]
	ctr.XORKeyStream(vData, vData)

	assert.Equal(t, data, vData)
}

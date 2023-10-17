package cryptostream

import (
	"crypto/aes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveKeyIV(t *testing.T) {
	pass := []byte("password")

	_, _, err := deriveKeyIV(pass, []byte{0x1})
	assert.Error(t, err, "invalid salt length")

	key, iv, err := deriveKeyIV(pass, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0})
	assert.NoError(t, err)
	assert.Len(t, iv, aes.BlockSize)
	assert.Len(t, key, 32) // AES256

	// Check reproducability
	k2, i2, err := deriveKeyIV(pass, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0})
	assert.NoError(t, err)
	assert.Equal(t, key, k2)
	assert.Equal(t, iv, i2)
}

func TestGetRandomSalt(t *testing.T) {
	salt, err := getRandomSalt()
	require.NoError(t, err)
	assert.Len(t, salt, saltLength)

	s2, err := getRandomSalt()
	require.NoError(t, err)
	assert.NotEqual(t, salt, s2, "SHOULD never happen")
}

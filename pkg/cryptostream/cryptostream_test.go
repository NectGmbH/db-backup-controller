package cryptostream

import (
	"crypto/aes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeaderLength(t *testing.T) {
	require.Equal(t, aes.BlockSize, len(header)+saltLength, "header + salt MUST have length of one block")
}

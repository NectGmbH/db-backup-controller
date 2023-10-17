package cryptostream

import (
	"crypto/aes"
	"io"

	"github.com/pkg/errors"
	"github.com/starius/aesctrat"
)

type (
	// CryptoWriteCloser implements an io.Writer following the
	// specification of this package onto a given io.Writer
	CryptoWriteCloser struct {
		aes *aesctrat.AesCtr
		iv  []byte

		next          io.Writer
		blocksWritten int
		buf           []byte
	}
)

var _ io.WriteCloser = (*CryptoWriteCloser)(nil)

// NewWriter creates a new CryptoWriteCloser, gets a random salt
// and writes the header to the underlying writer
func NewWriter(next io.Writer, pass []byte) (*CryptoWriteCloser, error) {
	// Put together our IV / Key from a salted pass
	salt, err := getRandomSalt()
	if err != nil {
		return nil, errors.Wrap(err, "getting random salt")
	}

	key, iv, err := deriveKeyIV(pass, salt)
	if err != nil {
		return nil, errors.Wrap(err, "deriving key/iv")
	}

	// Build our writer
	cw := &CryptoWriteCloser{
		aes: aesctrat.NewAesCtr(key),
		iv:  iv,

		next: next,
	}

	// Write the header to the underlying writer
	n, err := next.Write(append(header, salt...))
	if err != nil {
		return nil, errors.Wrap(err, "writing header")
	}
	if n != aes.BlockSize {
		return nil, errors.Errorf("wrote only %d / %d header bytes", n, aes.BlockSize)
	}

	// return everything
	return cw, nil
}

// Close implements the io.Closer interface and MUST be called after
// all writes are finished as the Write method might NOT have written
// all data to the underlying writer. This method ensures the data is
// fully and properly written
func (c *CryptoWriteCloser) Close() error {
	if len(c.buf) == 0 {
		// Nice! No remains, no issue!
		return nil
	}

	// We need to finish encrypting stuff
	c.aes.XORKeyStreamAt(c.buf, c.buf, c.iv, uint64(c.blocksWritten*aes.BlockSize))
	n, err := c.next.Write(c.buf)
	if err != nil {
		return errors.Wrap(err, "writing remaining data to underlying writer")
	}
	if n != len(c.buf) {
		return errors.Errorf("incomplete write to underlying writer %d/%d", n, len(c.buf))
	}

	return nil
}

// Write implements the io.Writer interface. See Close for hints how
// to properly write all data to the underlying writer. This method
// contains a buffer which buffers up to 16 bytes.
func (c *CryptoWriteCloser) Write(p []byte) (n int, err error) {
	data := append(c.buf, p...) //nolint:gocritic // This intentionally does NOT use the same slice

	// Get fully available blocks to write
	var (
		availBlocks = len(data) / aes.BlockSize
		encDataLen  = availBlocks * aes.BlockSize
	)

	// Encrypt the data
	c.aes.XORKeyStreamAt(data[:encDataLen], data[:encDataLen], c.iv, uint64(c.blocksWritten*aes.BlockSize))

	// Put full blocks into underlying writer
	wn, err := c.next.Write(data[:encDataLen])
	if err != nil {
		return n, errors.Wrap(err, "writing encrypted data to underlying writer")
	}
	if wn != encDataLen {
		return n, errors.Errorf("incomplete write to underlying writer %d/%d", wn, encDataLen)
	}

	// Buffer remains
	c.blocksWritten += availBlocks
	c.buf = data[encDataLen:]

	// Tell them we wrote all data they gave us
	return len(p), nil
}

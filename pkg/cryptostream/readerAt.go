package cryptostream

import (
	"bytes"
	"crypto/aes"
	"io"
	"math"

	"github.com/pkg/errors"
	"github.com/starius/aesctrat"
)

type (
	// CryptoReaderAt implements an io.ReaderAt on an AES-256-CTR
	// encrypted data stream. The data stream is expected to have the
	// format defined in this library.
	CryptoReaderAt struct {
		aes *aesctrat.AesCtr
		iv  []byte

		next io.ReaderAt
	}
)

var _ io.ReaderAt = CryptoReaderAt{}

// NewReaderAt creates a new CryptoReaderAt on the given io.ReaderAt,
// reads header and salt and verifies the stream follows the format
// specified in this library.
func NewReaderAt(next io.ReaderAt, pass []byte) (CryptoReaderAt, error) {
	// Validate this is a stream we can work on
	hdrBuf := make([]byte, len(header))
	_, err := next.ReadAt(hdrBuf, 0)
	if err != nil {
		return CryptoReaderAt{}, errors.Wrap(err, "reading header from stream")
	}
	if !bytes.Equal(hdrBuf, header) {
		return CryptoReaderAt{}, errors.New("stream does not have proper header")
	}

	// Get the salt from the stream
	salt := make([]byte, saltLength)
	n, err := next.ReadAt(salt, int64(len(header)))
	if err != nil {
		return CryptoReaderAt{}, errors.Wrap(err, "reading salt from stream")
	}
	if n != saltLength {
		return CryptoReaderAt{}, errors.Errorf("read %d of %d byte salt", n, saltLength)
	}

	// Create IV / Key from pass and salt
	key, iv, err := deriveKeyIV(pass, salt)
	if err != nil {
		return CryptoReaderAt{}, errors.Wrap(err, "deriving key/iv")
	}

	// return everything
	return CryptoReaderAt{
		aes: aesctrat.NewAesCtr(key),
		iv:  iv,

		next: next,
	}, nil
}

// ReadAt implements the io.ReaderAt interface
func (c CryptoReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	// We've been asked to read at a position {off} in the stream. What
	// they don't know: We've added {aes.BlockSize} bytes before that
	// therefore we need to shift that position by {aes.BlockSize}.
	// Afterwards there is the "issue" we always need to read full
	// blocks which is why we need to reduce the position to the
	// previous block start and read more than the given buffer can
	// deal with, decrypt the stuff and then pass them exactly the
	// part they asked for.

	var (
		// Increase the offset by one block (header)
		intOff = off + aes.BlockSize
		// Start read at the beginning of the block containing {intOff}
		readStartOff = intOff / aes.BlockSize * aes.BlockSize
		// End the read at a block boundary which fully fits into {p}
		readEndOff = (intOff + int64(len(p))) / aes.BlockSize * aes.BlockSize
	)

	if readEndOff == readStartOff {
		// We were asked to read less than one block, that doesn't make
		// any progress so we read one block to fill {p} fully
		readEndOff += aes.BlockSize
	}

	// Create a buffer to fit the data and read it
	data := make([]byte, readEndOff-readStartOff)
	n, err = c.next.ReadAt(data, readStartOff)
	switch {
	case err == nil:
		// That's fine and there is more data

	case errors.Is(err, io.EOF):
		// That's also fine but there is no more data

	default:
		// Well, that's not fine.
		return 0, errors.Wrap(err, "reading underlying data")
	}

	// Decrypt the data but don't take our header into account
	c.aes.XORKeyStreamAt(data, data, c.iv, uint64(readStartOff-aes.BlockSize))

	// Now give them the data - either the amount we read or the max
	// length of the buffer they passed in for us, which is less
	ePos := int(math.Min(float64(n), float64(intOff-readStartOff+int64(len(p)))))
	return copy(p, data[intOff-readStartOff:ePos]), nil
}

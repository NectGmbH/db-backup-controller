// Package cryptostream implements transparent encryption on ReaderAt
// and Writer streams in order to enforce encrypted backups
package cryptostream

import "crypto/aes"

const (
	cryptoIVLen      = 16
	cryptoKeyLen     = 32
	pbkdf2Iterations = 300000
	saltLength       = 8 // Do NOT change!
)

var header = []byte("DBCCrypt")

// HeaderSize represents the metadata size prepended to the data
// stream and needs to be subtracted when passing the payload size
// by using the file size of the encrypted data
const HeaderSize = aes.BlockSize

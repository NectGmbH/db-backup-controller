package cryptostream

import (
	"crypto/rand"
	"crypto/sha512"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pbkdf2"
)

func deriveKeyIV(pass, salt []byte) (key, iv []byte, err error) {
	if len(salt) != saltLength {
		return nil, nil, errors.Errorf("invalid salt length %d", len(salt))
	}

	rawKey := pbkdf2.Key(pass, salt, pbkdf2Iterations, cryptoIVLen+cryptoKeyLen, sha512.New)
	return rawKey[:cryptoKeyLen], rawKey[cryptoKeyLen : cryptoIVLen+cryptoKeyLen], nil
}

func getRandomSalt() (salt []byte, err error) {
	salt = make([]byte, saltLength)

	n, err := rand.Read(salt)
	if err != nil {
		return nil, errors.Wrap(err, "reading random bytes")
	}

	if n != saltLength {
		return nil, errors.Errorf("incomplete salt read %d / %d", n, saltLength)
	}

	return salt, nil
}

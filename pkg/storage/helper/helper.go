// Package helper contains simple utilities to help with storage management
package helper

import "io"

type (
	// ReaderAtCloser combines ReaderAt and ReadCloser interfaces
	ReaderAtCloser interface {
		io.ReadCloser
		io.ReaderAt
	}
)

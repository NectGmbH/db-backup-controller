package cockroach

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"sync"

	"github.com/pkg/errors"
)

const archiveFooterPadSize = 524288 // 512Ki

type (
	archiveFileInfo struct {
		StartOffset int64 `json:"o"`
		Size        int64 `json:"s"`
	}

	archiveFooter map[string]archiveFileInfo

	archiveReader struct {
		footer archiveFooter
		lock   sync.Mutex
		next   io.ReaderAt
	}

	archiveWriter struct {
		footer archiveFooter
		next   io.Writer

		openFile string
		written  int64
	}
)

// --- Footer

func (a *archiveFooter) DecodeFrom(r io.ReaderAt, rSize int64) error {
	raw, err := io.ReadAll(io.NewSectionReader(r, rSize-archiveFooterPadSize, archiveFooterPadSize))
	if err != nil {
		return errors.Wrap(err, "reading footer")
	}
	if n := len(raw); n != archiveFooterPadSize {
		return errors.Errorf("read only %d of %d bytes of footer", n, archiveFooterPadSize)
	}

	return errors.Wrap(
		json.NewDecoder(bytes.NewReader(bytes.TrimRight(raw, string([]byte{0x0})))).Decode(a),
		"decoding footer",
	)
}

func (a archiveFooter) EncodeTo(w io.Writer) (err error) {
	buf := new(bytes.Buffer)
	if err = json.NewEncoder(buf).Encode(a); err != nil {
		return errors.Wrap(err, "encoding footer")
	}

	if _, err = buf.Write(bytes.Repeat([]byte{0x0}, archiveFooterPadSize-buf.Len())); err != nil {
		return errors.Wrap(err, "padding footer")
	}

	_, err = buf.WriteTo(w)
	return errors.Wrap(err, "writing footer")
}

// --- Reader

func newArchiveReader(r io.ReaderAt, size int64) (*archiveReader, error) {
	a := &archiveReader{next: r}
	return a, errors.Wrap(a.footer.DecodeFrom(r, size), "getting footer")
}

func (a *archiveReader) Open(name string) (*io.SectionReader, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	info, ok := a.footer[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return io.NewSectionReader(a.next, info.StartOffset, info.Size), nil
}

// --- Writer

func newArchiveWriter(w io.Writer) *archiveWriter {
	return &archiveWriter{
		footer: make(archiveFooter),
		next:   w,
	}
}

func (a *archiveWriter) Close() error {
	a.closeIfOpen()

	return errors.Wrap(
		a.footer.EncodeTo(a.next),
		"writing footer",
	)
}

func (a *archiveWriter) Create(name string) error {
	a.closeIfOpen()

	if _, ok := a.footer[name]; ok {
		return fs.ErrExist
	}

	a.openFile = name
	a.footer[name] = archiveFileInfo{StartOffset: a.written}
	return nil
}

func (a *archiveWriter) Write(data []byte) (n int, err error) {
	if a.openFile == "" {
		return 0, errors.New("file not opened")
	}

	n, err = a.next.Write(data)
	if err != nil {
		return n, errors.Wrap(err, "writing to underlying writer")
	}

	a.written += int64(n)
	return n, nil
}

func (a *archiveWriter) closeIfOpen() {
	if a.openFile != "" {
		info := a.footer[a.openFile]
		info.Size = a.written - a.footer[a.openFile].StartOffset
		a.footer[a.openFile] = info
		a.openFile = ""
	}
}

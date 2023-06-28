package cockroach

import (
	"bytes"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveHeaderEncoding(t *testing.T) {
	var (
		buf       = new(bytes.Buffer)
		decFooter = archiveFooter{}
		footer    = archiveFooter{
			"myfile.txt": archiveFileInfo{StartOffset: archiveFooterPadSize, Size: 25},
		}
	)

	err := footer.EncodeTo(buf)
	require.NoError(t, err)

	assert.Equal(t, archiveFooterPadSize, buf.Len())
	assert.Equal(t, "{\"myfile.txt\":{\"o\":524288,\"s\":25}}\n", strings.TrimRight(buf.String(), string([]byte{0x0})))

	err = decFooter.DecodeFrom(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	assert.Equal(t, footer, decFooter)

	_, ok := footer["iamnothere.txt"]
	assert.False(t, ok)
}

func TestArchiveWriter(t *testing.T) {
	var (
		buf      = new(bytes.Buffer)
		aw       = newArchiveWriter(buf)
		testdata = []byte("I'm file content!")

		err error
	)

	require.NoError(t, aw.Create("test.txt"))
	assert.Equal(t, 0, buf.Len())

	_, err = aw.Write(testdata)
	assert.NoError(t, err)
	assert.Equal(t, testdata, buf.Bytes())

	assert.ErrorIs(t, aw.Create("test.txt"), fs.ErrExist, "must not allow duplicate file")
	_, err = aw.Write(testdata)
	assert.Error(t, err, "must not allow write when no file is open")

	require.NoError(t, aw.Create("anotherfile.txt"))
	_, err = aw.Write(testdata)
	assert.NoError(t, err)

	assert.NoError(t, aw.Close())
	assert.Equal(t, 2*len(testdata)+archiveFooterPadSize, buf.Len())

	assert.Equal(t, archiveFooter{
		"test.txt":        archiveFileInfo{Size: int64(len(testdata))},
		"anotherfile.txt": archiveFileInfo{Size: int64(len(testdata)), StartOffset: int64(len(testdata))},
	}, aw.footer)
}

func TestArchiveReader(t *testing.T) {
	var (
		buf      = new(bytes.Buffer)
		testdata = []byte("I'm file content!")

		err error
	)

	_, err = buf.Write(append(testdata, testdata...))
	require.NoError(t, err)

	err = archiveFooter{
		"myfile.txt":      archiveFileInfo{Size: int64(len(testdata))},
		"anotherfile.txt": archiveFileInfo{Size: int64(len(testdata)), StartOffset: int64(len(testdata))},
	}.EncodeTo(buf)
	require.NoError(t, err)

	ar, err := newArchiveReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	sr, err := ar.Open("myfile.txt")
	require.NoError(t, err, "must open existing file")
	raw, err := io.ReadAll(sr)
	assert.NoError(t, err, "must read contents from existing file")
	assert.Equal(t, testdata, raw, "existing file must have expected content")

	sr, err = ar.Open("anotherfile.txt")
	require.NoError(t, err, "must open existing file")
	raw, err = io.ReadAll(sr)
	assert.NoError(t, err, "must read contents from existing file")
	assert.Equal(t, testdata, raw, "existing file must have expected content")

	_, err = ar.Open("iamcertainlynothere.txt")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

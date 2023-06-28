package cockroach

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type (
	backupReader struct {
		r io.ReaderAt

		logger *logrus.Entry
		ar     *archiveReader
	}
)

func newBackupReader(source io.ReaderAt, size int64, logger *logrus.Entry) (*backupReader, error) {
	if logger == nil {
		l := logrus.New()
		l.SetOutput(io.Discard)
		logger = l.WithContext(context.Background())
	}

	ar, err := newArchiveReader(source, size)
	if err != nil {
		return nil, errors.Wrap(err, "opening archive reader")
	}

	return &backupReader{
		r:      source,
		logger: logger,
		ar:     ar,
	}, nil
}

// ServeHTTP implements http.Handler and acts as a http filesystem
func (b backupReader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn := strings.TrimPrefix(r.URL.Path, "/crdb-backup/")
	b.logger.WithField("range", r.Header.Get("range")).WithField("path", fn).Debug("got request for file")

	if r.Method != http.MethodGet {
		http.Error(w, "this is a read-only endpoint", http.StatusMethodNotAllowed)
		return
	}

	start, end, isFull, err := b.parseRange(r)
	if err != nil {
		b.logger.WithError(err).Error("parsing range header")
		http.Error(w, errors.Wrap(err, "parsing range header").Error(), http.StatusBadRequest)
		return
	}

	f, err := b.ar.Open(fn)
	switch {
	case err == nil:
		// Cool, handle below

	case errors.Is(err, fs.ErrNotExist):
		http.Error(w, "that's not the file you're looking for", http.StatusNotFound)
		return

	default:
		b.logger.WithError(err).Error("opening file from archive")
		http.Error(w, errors.Wrap(err, "opening file from archive").Error(), http.StatusInternalServerError)
		return
	}

	if end == -1 {
		// We didn't know what the end is but the end should be the absolute end:
		end = f.Size()
	}

	// In all cases: We should tell them we accept ranges...
	w.Header().Set("Accept-Ranges", "bytes")

	if isFull {
		// Easy: Just throw the whole stuff at them, don't annotate any special headers
		w.Header().Set("Content-Length", strconv.FormatInt(f.Size(), 10))
		w.WriteHeader(http.StatusOK)

		if _, err = io.Copy(w, f); err != nil {
			b.logger.WithError(err).Error("copying file")
		}
		return
	}

	// Well, frick: They want a range. :(

	// Lets get our chunk to send
	chunk := io.NewSectionReader(f, start, end)

	// Now annotate the response with appropriate headers:
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, f.Size()))
	w.Header().Set("Content-Length", strconv.FormatInt(end-start, 10))
	w.WriteHeader(http.StatusPartialContent)

	// Now throw the requested data at them:
	if _, err = io.Copy(w, chunk); err != nil {
		b.logger.WithError(err).Error("copying chunk")
	}
}

func (backupReader) parseRange(r *http.Request) (start, end int64, isFull bool, err error) {
	// Get the header
	rh := r.Header.Get("range")
	if rh == "" {
		// No range requested, do a full-copy
		return 0, -1, true, nil
	}

	// Check whether we got a byte range (only supported range format)
	if !strings.HasPrefix(rh, "bytes=") {
		return 0, 0, false, errors.New("unsupported range unit")
	}

	// Extract ranges and check whether exactly one was requested
	rh = strings.TrimPrefix(rh, "bytes=")
	ranges := strings.Split(rh, ",")

	if len(ranges) > 1 {
		return 0, 0, false, errors.New("multi-range read is unsupported")
	}

	// Parse the range into two values what to read
	se := strings.Split(ranges[0], "-")
	if len(se) != 2 { //nolint:gomnd
		return 0, 0, false, errors.New("unexpected range format")
	}

	if se[0] == "" {
		start = 0
	} else {
		if start, err = strconv.ParseInt(se[0], 10, 64); err != nil {
			return 0, 0, false, errors.Wrap(err, "parsing range start")
		}
	}

	if se[1] == "" {
		end = -1
	} else {
		if end, err = strconv.ParseInt(se[1], 10, 64); err != nil {
			return 0, 0, false, errors.Wrap(err, "parsing range end")
		}
	}

	return start, end, start == 0 && end == -1, nil
}

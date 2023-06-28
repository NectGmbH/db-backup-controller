package cockroach

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type (
	backupWriter struct {
		w io.Writer

		logger  *logrus.Entry
		memFS   map[string][]byte
		reqLock sync.Mutex
		aw      *archiveWriter
	}
)

func newBackupWriter(w io.Writer, logger *logrus.Entry) *backupWriter {
	if logger == nil {
		l := logrus.New()
		l.SetOutput(io.Discard)
		logger = l.WithContext(context.Background())
	}

	return &backupWriter{
		w:      w,
		logger: logger,
		memFS:  make(map[string][]byte),
		aw:     newArchiveWriter(w),
	}
}

// Close closes the ZIP writer and afterwards the writing end of the
// pipe after it copied the remaining files from the memFS into the
// ZIP writer
func (b *backupWriter) Close() (err error) {
	b.reqLock.Lock()
	defer b.reqLock.Unlock()

	for fn, data := range b.memFS {
		if err := b.aw.Create(fn); err != nil {
			return errors.Wrap(err, "creating header for meta-file")
		}
		if _, err = io.Copy(b.aw, bytes.NewReader(data)); err != nil {
			return errors.Wrap(err, "writing meta-file")
		}
	}

	return errors.Wrap(b.aw.Close(), "closing archive")
}

// ServeHTTP implements http.Handler and acts as a http filesystem
//
//nolint:funlen,gocyclo // Makes no sense to shorten 2 lines
func (b *backupWriter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ZIP writer cannot handle more than one file at once, lets ensure
	// it doesn't choke on multiple requests. (Also our memFS is a map
	// which would just explode…)
	b.reqLock.Lock()
	defer b.reqLock.Unlock()

	var (
		cLen    int64
		err     error
		fn      = strings.TrimPrefix(r.URL.Path, "/crdb-backup/")
		fLogger = b.logger.WithField("filename", fn)
	)

	if r.Header.Get("Content-Length") != "" {
		cLen, err = strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			http.Error(w, errors.Wrap(err, "reading content length").Error(), http.StatusBadRequest)
			return
		}
	}

	switch {
	case strings.HasSuffix(fn, ".sst"):
		// Backup-data, might be HUGE, not supposed to be deleted, goes to ZIP
		if r.Method != http.MethodPut {
			http.Error(w, fmt.Sprintf("not sure what %s should do", r.Method), http.StatusMethodNotAllowed)
			return
		}

		if err := b.aw.Create(fn); err != nil {
			fLogger.WithError(err).Error("creating archive file")
			http.Error(w, errors.Wrap(err, "creating archive file").Error(), http.StatusInternalServerError)
			return
		}

		n, err := io.Copy(b.aw, r.Body)
		if err != nil {
			fLogger.WithError(err).Error("writing file to archive")
			http.Error(w, errors.Wrap(err, "writing file to archive").Error(), http.StatusInternalServerError)
			return
		}

		if cLen > 0 && n != cLen {
			fLogger.Error("did not copy full content length")
			http.Error(w, fmt.Sprintf("read only %d of %d byte", n, cLen), http.StatusInternalServerError)
			return
		}

		fLogger.WithField("size", cLen).Debug("added file to archive")

		w.WriteHeader(http.StatusCreated)

	case strings.HasPrefix(fn, "BACKUP") || strings.HasPrefix(fn, "progress/BACKUP"):
		// Meta-file, supposedly small, can be deleted, goes to memFS
		switch r.Method {
		case http.MethodDelete:
			// Okay, lets forget about it
			delete(b.memFS, fn)
			w.WriteHeader(http.StatusNoContent)

		case http.MethodGet:
			// If we have it: Lets return it
			if _, ok := b.memFS[fn]; !ok {
				// What's that? Give first, read then!
				http.Error(w, "you didn't send that", http.StatusNotFound)
				return
			}

			if _, err = w.Write(b.memFS[fn]); err != nil {
				fLogger.WithError(err).Error("serving file from memFS")
				return
			}

		case http.MethodPut:
			// Nice, new data!
			if b.memFS[fn], err = io.ReadAll(r.Body); err != nil {
				http.Error(w, errors.Wrap(err, "reading body").Error(), http.StatusBadRequest)
				return
			}

			if n := int64(len(b.memFS[fn])); cLen > 0 && n != cLen {
				fLogger.Error("did not copy full content length")
				http.Error(w, fmt.Sprintf("read only %d of %d byte", n, cLen), http.StatusInternalServerError)
				return
			}

			fLogger.WithField("size", cLen).Debug("added file to memFS")

			w.WriteHeader(http.StatusCreated)

		default:
			// Ehm. No.
			http.Error(w, fmt.Sprintf("not sure what %s should do", r.Method), http.StatusMethodNotAllowed)
			return
		}

	default:
		// Ehm. Well. No thanks?
		http.Error(w, fmt.Sprintf("not sure what to do with %s", fn), http.StatusBadRequest)
		return
	}
}

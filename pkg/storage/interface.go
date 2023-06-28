// Package storage describes how to interact with underlying storage
package storage

import (
	"context"
	"errors"
	"io"
	"time"

	v1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
	"github.com/NectGmbH/db-backup-controller/pkg/storage/helper"
	"github.com/NectGmbH/db-backup-controller/pkg/storage/s3"
)

type (
	// Manager describes the interface all storage implementations have
	// to follow to be usable in the backup-runner
	Manager interface {
		// CleanupBackups takes care of removing expired backups from the
		// remote storage
		CleanupBackups(context.Context) error
		// DownloadAsReader fetches the given backup (must exist) and
		// returns an io.ReadCloser for it
		DownloadAsReader(ctx context.Context, name string) (helper.ReaderAtCloser, int64, error)
		// DownloadToFile fetches the given backup (must exist) and saves
		// it to the given targetPath
		DownloadToFile(ctx context.Context, name, targetPath string) error
		// DownloadPITBackupAsReader takes a Point-in-Time, fetches the
		// closest older backup to that point-in-time and returns an
		// io.ReadCloser for it
		DownloadPITBackupAsReader(ctx context.Context, pit time.Time) (helper.ReaderAtCloser, int64, error)
		// DownloadPITBackupToFile takes a Point-in-Time and downloads the
		// closest older backup to that point-in-time to the given path
		DownloadPITBackupToFile(ctx context.Context, pit time.Time, targetPath string) error
		// ListAvailableBackups fetches a list of backups stored on the
		// remote storage and returns the names suitable for DownloadToFile
		ListAvailableBackups(ctx context.Context) ([]string, error)
		// UploadFromFile takes a local file and uploads the contents under
		// the filename the file on the filesystem has
		UploadFromFile(ctx context.Context, filePath string) error
		// UploadFromReader takes a name and a reader to upload a backup
		// to the remote storage. The name is used as storage name and
		// later available in ListAvailableBackups and DownloadToFile
		UploadFromReader(ctx context.Context, name string, data io.Reader, size int64) error
	}
)

// ErrStorageEngineUnknown means you've requested a storage engine
// not defined in the registry
var ErrStorageEngineUnknown = errors.New("specified storage engine is unknown")

// New creates a new preconfigured instance of the desired storage engine
func New(ctx context.Context, storageLocation *v1.DatabaseBackupStorageLocation, cfg *v1.DatabaseBackup) (Manager, error) { //nolint:ireturn,lll // This is a registry
	switch storageLocation.StorageType {
	case "s3":
		return s3.New(ctx, storageLocation, cfg) //nolint:wrapcheck // It's fine to return the error unwrapped as this is only a registry

	default:
		return nil, ErrStorageEngineUnknown
	}
}

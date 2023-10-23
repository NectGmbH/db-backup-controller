// Package s3 provides a storage.Manager for S3
package s3

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"

	v1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
	"github.com/NectGmbH/db-backup-controller/pkg/labelmanager"
	"github.com/NectGmbH/db-backup-controller/pkg/storage/helper"
)

const (
	labelmanagerStorageFileName = ".labels"
	minioMaxIdleConns           = 10
	minioIdleConnTimeout        = 30 * time.Second
	minioTLSHandshakeTimeout    = 30 * time.Second
)

type (
	// Storage implements the storage.Manager interface to provide
	// backup storage access in S3 compatible (S3, GCS, MinIO) storages
	Storage struct {
		client *minio.Client

		config          *v1.DatabaseBackupSpec
		storageLocation *v1.DatabaseBackupStorageLocation

		storagePath string

		labels *labelmanager.Manager
	}
)

// New creates a new S3 Storage
func New(ctx context.Context, storageLocation *v1.DatabaseBackupStorageLocation, cfg *v1.DatabaseBackup) (*Storage, error) {
	minioOpts := &minio.Options{
		Creds:  credentials.NewStaticV4(storageLocation.StorageAccessKeyID.Value, storageLocation.StorageSecretAccessKey.Value, ""),
		Secure: storageLocation.StorageUseSSL,
		Region: storageLocation.StorageLocation,
	}

	if storageLocation.StorageInsecureSkipVerify {
		minioOpts.Transport = &http.Transport{
			MaxIdleConns:       minioMaxIdleConns,
			IdleConnTimeout:    minioIdleConnTimeout,
			DisableCompression: true,
			//#nosec:G402 // That's exactly the intention of this code path
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout: minioTLSHandshakeTimeout,
		}
	}

	client, err := minio.New(storageLocation.StorageEndpoint, minioOpts)
	if err != nil {
		return nil, errors.Wrap(err, "creating minio client")
	}

	stor := &Storage{
		client: client,

		config:          &cfg.Spec,
		storageLocation: storageLocation,

		storagePath: strings.Join([]string{cfg.Namespace, cfg.Name}, "-"),
	}

	if cfg.Spec.UseSingleBackupTarget {
		return stor, nil
	}

	stor.labels, err = stor.loadLabelManager(ctx)
	return stor, errors.Wrap(err, "loading label manager storage")
}

// CleanupBackups takes care of removing expired backups from the
// remote storage
func (s Storage) CleanupBackups(ctx context.Context) (err error) {
	if s.config.UseSingleBackupTarget {
		// We don't clean that up.
		return nil
	}

	// Let the label manager clean itself up
	s.labels.CleanRetentions()

	// Now we get all entries which should no longer exist and make sure they don't
	for _, entry := range s.labels.GetUnretainedEntries() {
		if err = s.client.RemoveObject(
			ctx,
			s.storageLocation.StorageBucket,
			path.Join(s.storagePath, entry),
			minio.RemoveObjectOptions{},
		); err != nil {
			return errors.Wrap(err, "deleting expired backup")
		}

		// We've gotten rid of it, remove the knowledge about it
		s.labels.Remove(entry)
	}

	for _, entry := range s.labels.GetRetainedEntries() {
		_, err := s.client.StatObject(
			ctx,
			s.storageLocation.StorageBucket,
			path.Join(s.storagePath, entry),
			minio.StatObjectOptions{},
		)
		if err == nil {
			// We got a stat, that object is there, skip
			continue
		}

		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			// Well, that backup is for sure gone...
			s.labels.Remove(entry)
			continue
		}
		return errors.Wrap(err, "fetching object stats for backup")
	}

	// And finally we store the state back to the bucket
	return errors.Wrap(
		s.saveLabelManager(ctx),
		"storing label manager content",
	)
}

// DownloadAsReader fetches the given backup (must exist) and
// returns an io.ReadCloser for it
func (s Storage) DownloadAsReader(ctx context.Context, name string) (helper.ReaderAtCloser, int64, error) { //nolint:ireturn,lll // Interface is expecting this
	sourceName := path.Join(s.storagePath, name)
	if s.config.UseSingleBackupTarget {
		sourceName = s.storagePath
	}

	obj, err := s.client.GetObject(ctx, s.storageLocation.StorageBucket, sourceName, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, errors.Wrap(err, "getting stored object")
	}

	// GetObject does not error when object does not exist, Stat does.
	stat, err := obj.Stat()
	if err != nil {
		return nil, 0, errors.Wrap(err, "getting stored object stat")
	}

	return obj, stat.Size, nil
}

// DownloadToFile fetches the given backup (must exist) and saves
// it to the given targetPath
func (s Storage) DownloadToFile(ctx context.Context, name, targetPath string) error {
	f, err := os.Create(targetPath) //#nosec:G304 // This library is intended to write to passed file
	if err != nil {
		return errors.Wrap(err, "creating target file")
	}
	defer f.Close() //nolint:errcheck // This might leak FDs but this is a library and should not log

	obj, _, err := s.DownloadAsReader(ctx, name)
	if err != nil {
		return errors.Wrap(err, "getting reader for objectg")
	}
	defer obj.Close() //nolint:errcheck // This might leak FDs but this is a library and should not log

	_, err = io.Copy(f, obj)
	return errors.Wrap(err, "copying object to target file")
}

// DownloadPITBackupAsReader takes a Point-in-Time, fetches the
// closest older backup to that point-in-time and returns an
// io.ReadCloser for it
func (s Storage) DownloadPITBackupAsReader(ctx context.Context, pit time.Time) (helper.ReaderAtCloser, int64, error) { //nolint:ireturn,lll // Interface is expecting this
	if s.config.UseSingleBackupTarget {
		// There is no point in time, take the object or don't
		return s.DownloadAsReader(ctx, "")
	}

	backup, err := s.labels.GetClosestOlderBackup(pit)
	if err != nil {
		return nil, 0, errors.Wrap(err, "finding backup for given time")
	}

	return s.DownloadAsReader(ctx, backup)
}

// DownloadPITBackupToFile takes a Point-in-Time and downloads the
// closest older backup to that point-in-time to thhe given path
func (s Storage) DownloadPITBackupToFile(ctx context.Context, pit time.Time, targetPath string) error {
	if s.config.UseSingleBackupTarget {
		// There is no point in time, take the object or don't
		return s.DownloadToFile(ctx, "", targetPath)
	}

	backup, err := s.labels.GetClosestOlderBackup(pit)
	if err != nil {
		return errors.Wrap(err, "finding backup for given time")
	}

	return s.DownloadToFile(ctx, backup, targetPath)
}

// ListAvailableBackups fetches a list of backups stored on the
// remote storage and returns the names suitable for DownloadToFile
func (s Storage) ListAvailableBackups(ctx context.Context) ([]string, error) {
	if s.config.UseSingleBackupTarget {
		// Special case: There can be only one - or none - lets see what's the case
		info, err := s.client.StatObject(ctx, s.storageLocation.StorageBucket, s.storagePath, minio.StatObjectOptions{})
		if err != nil {
			var mErr minio.ErrorResponse
			if ok := errors.As(err, &mErr); ok && mErr.StatusCode == http.StatusNotFound {
				// There is no backup but that's fine
				return nil, nil
			}

			return nil, errors.Wrap(err, "checking object existence")
		}
		return []string{info.Key}, nil
	}

	// Not a single target backup, lets ask the label manager
	return s.labels.GetRetainedEntries(), nil
}

// UploadFromFile takes a local file and uploads the contents under
// the filename the file on the filesystem has
func (s Storage) UploadFromFile(ctx context.Context, filePath string) error {
	f, err := os.Open(filePath) //#nosec:G304 // This library is intended to work with the given file
	if err != nil {
		return errors.Wrap(err, "opening source file")
	}
	defer f.Close() //nolint:errcheck // This might leak FDs but this is a library and should not log

	stat, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, "getting file-stat")
	}

	return s.UploadFromReader(ctx, path.Base(filePath), f, stat.Size())
}

// UploadFromReader takes a name and a reader to upload a backup
// to the remote storage. The name is used as storage name and
// later available in ListAvailableBackups and DownloadToFile
func (s Storage) UploadFromReader(ctx context.Context, name string, data io.Reader, size int64) (err error) {
	if err = s.uploadFromReader(ctx, name, data, size); err != nil {
		return err
	}

	if s.config.UseSingleBackupTarget {
		// Not using label manager
		return nil
	}

	// Add the new file to the label manager
	if err = s.labels.Add(name); err != nil {
		return errors.Wrap(err, "adding to label manager")
	}

	// And finally we store the state back to the bucket
	return errors.Wrap(
		s.saveLabelManager(ctx),
		"storing label manager content",
	)
}

func (s Storage) loadLabelManager(ctx context.Context) (*labelmanager.Manager, error) {
	var content io.Reader
	obj, err := s.client.GetObject(
		ctx,
		s.storageLocation.StorageBucket,
		path.Join(s.storagePath, labelmanagerStorageFileName),
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, errors.Wrap(err, "fetching label storage object")
	}

	// GetObject does not error when object does not exist, Stat does.
	if _, err = obj.Stat(); err != nil {
		var mErr minio.ErrorResponse
		if ok := errors.As(err, &mErr); ok && mErr.Code == "NoSuchKey" {
			goto parseAndReturn
		}

		return nil, errors.Wrap(err, "getting label storage object stat")
	}

	content = obj

parseAndReturn:
	lm, err := labelmanager.New(content, s.config.RetentionConfig)
	return lm, errors.Wrap(err, "initializing label manager")
}

func (s Storage) saveLabelManager(ctx context.Context) (err error) {
	labelContent := new(bytes.Buffer)
	if err = s.labels.Save(labelContent); err != nil {
		return errors.Wrap(err, "serializing label manager content")
	}

	return errors.Wrap(
		s.uploadFromReader(ctx, labelmanagerStorageFileName, labelContent, int64(labelContent.Len())),
		"storing label manager content",
	)
}

//revive:disable-next-line:confusing-naming // That's the implementation, naming is intended to be the same
func (s Storage) uploadFromReader(ctx context.Context, name string, data io.Reader, size int64) error {
	targetName := path.Join(s.storagePath, name)
	if s.config.UseSingleBackupTarget {
		targetName = s.storagePath
	}

	_, err := s.client.PutObject(ctx, s.storageLocation.StorageBucket, targetName, data, size, minio.PutObjectOptions{})
	if err != nil {
		return errors.Wrap(err, "uploading object")
	}

	return nil
}

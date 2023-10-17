package main

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	v1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
	"github.com/NectGmbH/db-backup-controller/pkg/backupengine"
	"github.com/NectGmbH/db-backup-controller/pkg/cryptostream"
	"github.com/NectGmbH/db-backup-controller/pkg/storage"
	"github.com/NectGmbH/db-backup-controller/pkg/storage/helper"
)

func executeRestore(restoreMode, backupID string) (err error) {
	// Can be asked to restore a backup
	// * Downloads backup (=> ./pkg/storage/...)
	// * Askes engine to restore that backup (=> ./pkg/backupengine/...)
	// * Engine knows what to execute to restore $db to $host with $credentials from file

	for i := range configStorage.BackupLocations {
		loc := configStorage.BackupLocations[i]

		logger := logrus.WithField("location", loc.StorageEndpoint)
		logger.Info("preparing restore")

		if err := restoreForLocation(engine, restoreMode, &loc, backupID); err != nil {
			logger.WithError(err).Error("restoring from location")
			continue
		}

		logger.Info("restore completed")
		return nil
	}

	return errors.New("no backup found to restore")
}

func restoreForLocation(engine backupengine.Implementation, restoreMode string, loc *v1.DatabaseBackupStorageLocation, backupID string) error {
	stor, err := storage.New(context.Background(), loc, &configBackup)
	if err != nil {
		return errors.Wrap(err, "getting storage provider")
	}

	var (
		r    helper.ReaderAtCloser
		size int64
	)
	switch restoreMode {
	case "name":
		if r, size, err = stor.DownloadAsReader(context.Background(), backupID); err != nil {
			return errors.Wrap(err, "getting specified backup")
		}

	case "point-in-time":
		t, err := time.Parse(time.RFC3339, backupID)
		if err != nil {
			return errors.Wrap(err, "parsing point-in-time for RFC3339")
		}

		if r, size, err = stor.DownloadPITBackupAsReader(context.Background(), t); err != nil {
			return errors.Wrap(err, "getting backup for point-in-time")
		}

	default:
		return errors.Errorf("invalid restore-mode %q", restoreMode)
	}
	defer func() {
		if err := r.Close(); err != nil {
			logrus.WithError(err).Error("closing backup (leaked fd)")
		}
	}()

	var (
		backupSrc  io.ReaderAt = r
		backupSize             = size
	)

	if loc.EncryptionPass.Value != "" {
		cryptR, err := cryptostream.NewReaderAt(r, []byte(loc.EncryptionPass.Value))
		if err != nil {
			return errors.Wrap(err, "creating crypto-reader")
		}
		backupSrc = cryptR
		backupSize = size - cryptostream.HeaderSize
	}

	if err = engine.RestoreBackup(backupSrc, backupSize); err != nil {
		return errors.Wrap(err, "restoring backup")
	}

	return nil
}

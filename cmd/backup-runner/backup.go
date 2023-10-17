package main

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/NectGmbH/db-backup-controller/pkg/cryptostream"
	"github.com/NectGmbH/db-backup-controller/pkg/storage"
)

func executeBackup() (err error) {
	// * Asks requested engine to take a backup (=> ./pkg/backupengine/...)
	// 	* Engine knows what to execute to backup $db from $host with $credentials
	// 	* Engine stores backup to file location it is given by runner
	// * Uploads backup to storage location (=> ./pkg/storage/...)
	// 	* Upload location in the bucket is a generated name from the backup definition name and the namespace
	// * Takes notes which backups exist, manages "labels" for them, if no more labels are attached removes backup
	// 	* Can run in "single backup" mode: No labels, no management, no retention, just a single uploaded target

	backupName := time.Now().UTC().Format("2006-01-02T15-04-05")

	for i := range configStorage.BackupLocations {
		loc := configStorage.BackupLocations[i]

		logger := logrus.WithFields(logrus.Fields{
			"backup":   backupName,
			"location": loc.StorageEndpoint,
		})
		logger.Info("preparing backup")

		logger.Debug("engine initialized")

		stor, err := storage.New(context.Background(), &loc, &configBackup)
		if err != nil {
			return errors.Wrap(err, "getting storage provider")
		}

		logger.Debug("storage initialized")

		logger.Info("starting backup")

		r, w := io.Pipe()

		go func() {
			if err = w.CloseWithError(func(w io.Writer) (err error) {
				var (
					backupDest = w
					cryptW     *cryptostream.CryptoWriteCloser
				)

				if loc.EncryptionPass.Value != "" {
					cryptW, err = cryptostream.NewWriter(w, []byte(loc.EncryptionPass.Value))
					if err != nil {
						return errors.Wrap(err, "creating crypto-writer")
					}
					backupDest = cryptW
				}

				if err = engine.CreateBackup(backupDest); err != nil {
					return errors.Wrap(err, "creating backup")
				}

				if cryptW == nil {
					return nil
				}

				return errors.Wrap(cryptW.Close(), "closing crypto writer")
			}(w)); err != nil {
				logger.WithError(err).Error("closing backup pipe")
			}
		}()

		if err = stor.UploadFromReader(context.Background(), backupName, r, -1); err != nil {
			return errors.Wrap(err, "uploading backup to storage location")
		}

		// Trigger backup cleanup in background
		go func() {
			if err := stor.CleanupBackups(context.Background()); err != nil {
				logger.WithError(err).Error("executing storage cleanup")
			}
		}()

		logger.Info("backup completed")
	}

	return nil
}

func nextExecutionFromCron(spec string) (time.Time, error) {
	cs, err := cron.ParseStandard(spec)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "parsing cron-spec")
	}

	return cs.Next(time.Now()), nil
}

func nextExecutionFromInterval(hours int64) (time.Time, error) {
	var (
		interval = time.Duration(hours) * time.Hour
		now      = time.Duration(time.Now().UnixNano())
	)

	wait := interval - (now % interval)
	return time.Now().Add(wait), nil
}

func tickAutoBackup(ticker chan struct{}) (err error) {
	// * Waits for time to come
	// * Pings ticker to start backup
	// * Goto "wait" and repeat

	for {
		var nextExecution time.Time

		switch {
		case configBackup.Spec.BackupCron != "":
			if nextExecution, err = nextExecutionFromCron(configBackup.Spec.BackupCron); err != nil {
				return errors.Wrap(err, "calculating next execution from cron")
			}

		case configBackup.Spec.BackupIntervalHours != 0:
			if nextExecution, err = nextExecutionFromInterval(configBackup.Spec.BackupIntervalHours); err != nil {
				return errors.Wrap(err, "calculating next execution from interval")
			}

		default:
			// The heck?
			return errors.New("neither cron nor interval given")
		}

		logrus.WithField("next_execution", nextExecution.Format(time.RFC3339)).Info("waiting for next execution")
		time.Sleep(time.Until(nextExecution))

		ticker <- struct{}{}
	}
}

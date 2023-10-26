package main

import (
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/Luzifer/rconfig/v2"

	"github.com/NectGmbH/db-backup-controller/pkg/backupengine"
	"github.com/NectGmbH/db-backup-controller/pkg/cryptostream"
)

const backupDestPrivileges = 0o700

var (
	cfg = struct {
		BackupEngine   string `flag:"backup-engine,b" default:"" description:"Which engine to use for unpacking the backup (MUST match the engine creating the backup)" validate:"nonzero"` //nolint:lll
		DestDir        string `flag:"dest-dir,d" default:"" description:"Where to unpack the backup (MUST NOT exist)" validate:"nonzero"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		Passphrase     string `flag:"passphrase,p" default:"" description:"Specify when unpacking an encrypted backup"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func initApp() error {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		return errors.Wrap(err, "parsing cli options")
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return errors.Wrap(err, "parsing log-level")
	}
	logrus.SetLevel(l)

	return nil
}

func main() {
	var err error
	if err = initApp(); err != nil {
		logrus.WithError(err).Fatal("initializing app")
	}

	if cfg.VersionAndExit {
		logrus.WithField("version", version).Info("backup-unpack")
		os.Exit(0)
	}

	if len(rconfig.Args()) == 1 {
		logrus.Fatalf("usage: %s [options] <backup file>", path.Base(rconfig.Args()[0]))
	}

	if _, err = os.Stat(cfg.DestDir); !errors.Is(err, fs.ErrNotExist) {
		logrus.Fatal("destination already exists")
	}

	engine := backupengine.GetByName(cfg.BackupEngine)
	if engine == nil {
		logrus.Fatal("unknown backup-engine specified")
	}

	backup, err := os.Open(rconfig.Args()[1])
	if err != nil {
		logrus.WithError(err).Fatal("opening backup")
	}
	defer backup.Close() //nolint:errcheck // The file is auto-closed by program exit

	stat, err := backup.Stat()
	if err != nil {
		logrus.WithError(err).Fatal("getting backup stat")
	}

	if err = os.MkdirAll(cfg.DestDir, backupDestPrivileges); err != nil {
		logrus.WithError(err).Fatal("creating backup destination")
	}

	var (
		backupReaderAt io.ReaderAt = backup
		backupSize                 = stat.Size()
	)
	if cfg.Passphrase != "" {
		if backupReaderAt, err = cryptostream.NewReaderAt(backupReaderAt, []byte(cfg.Passphrase)); err != nil {
			logrus.WithError(err).Fatal("opening crypto-reader")
		}
		backupSize -= cryptostream.HeaderSize
	}

	if err = engine.Unpack(backupReaderAt, backupSize, cfg.DestDir); err != nil {
		logrus.WithError(err).Fatal("unpacking backup")
	}

	logrus.Info("backup was successfully unpacked")
}

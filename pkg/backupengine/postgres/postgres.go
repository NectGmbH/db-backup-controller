// Package postgres contains the implementation of the backupengine
// for PostgreSQL
package postgres

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"

	backupControllerV1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
	"github.com/NectGmbH/db-backup-controller/pkg/backupengine/base"
	"github.com/NectGmbH/db-backup-controller/pkg/backupengine/opts"
)

type (
	// Engine implements backupengine interface
	Engine struct {
		baseEngine base.Engine
		spec       backupControllerV1.DatabaseBackupSpec
	}
)

// New creates a new Engine instance
func New() *Engine { return &Engine{} }

// CreateBackup is used to instruct the backup engine to create
// a backup. The means of doing so depends on the engine itself.
func (e *Engine) CreateBackup(w io.Writer) error {
	// https://www.postgresql.org/docs/current/app-pgdump.html
	//#nosec:G204 // Backing up the user-specified db is intentional
	cmd := exec.Command(
		"pg_dump",
		"--create",       // Begin the output with a command to create the database itself and reconnect to the created database.
		"--format=plain", // Use a plain SQL file
		e.spec.Postgres.Database,
	)

	cmd.Env = []string{
		fmt.Sprintf("PGHOST=%s", e.spec.Postgres.Host),
		fmt.Sprintf("PGPORT=%d", e.spec.Postgres.Port),
		fmt.Sprintf("PGUSER=%s", e.spec.Postgres.User),
		fmt.Sprintf("PGPASSWORD=%s", e.spec.Postgres.Pass.Value),
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = w

	return errors.Wrap(cmd.Run(), "running pg_dump")
}

// GetPodSpec generates a pod-spec from the given backup
// specificiation containing required volume mounts from secrets
// or envFrom definitions (and possible other special cases).
// The mounted secret will be added by the controller.
func (e *Engine) GetPodSpec(imagePrefix string) (coreV1.PodSpec, error) {
	podSpec, err := e.baseEngine.GetPodSpec()
	if err != nil {
		return podSpec, errors.Wrap(err, "getting base spec")
	}

	// Set postgres image
	podSpec.Containers[0].Image = strings.Join([]string{imagePrefix, "postgres", e.spec.DatabaseVersion}, "-")

	return podSpec, nil
}

// Init is called once per backup engine and allows to execute
// one-shot initialization tasks like registering new HTTP
// handlers
func (e *Engine) Init(options opts.InitOpts) error {
	if options.Spec.Postgres == nil {
		return errors.New("postgres config not available")
	}

	if err := e.baseEngine.Init(options); err != nil {
		return errors.Wrap(err, "initializing base engine")
	}

	e.spec = options.Spec

	return nil
}

// RestoreBackup receives an io.ReaderAt with the contents of
// the backup to be restored and the size of the backup. The
// means of doing so depends on the engine itself. The contents
// of the reader will be the same the engine provided during
// the CreateBackup result
func (e *Engine) RestoreBackup(r io.ReaderAt, size int64) error {
	cmd := exec.Command("psql", "-v", "ON_ERROR_STOP=1")

	cmd.Env = []string{
		fmt.Sprintf("PGHOST=%s", e.spec.Postgres.Host),
		fmt.Sprintf("PGPORT=%d", e.spec.Postgres.Port),
		fmt.Sprintf("PGUSER=%s", e.spec.Postgres.User),
		fmt.Sprintf("PGPASSWORD=%s", e.spec.Postgres.Pass.Value),
	}

	cmd.Stdin = io.NewSectionReader(r, 0, size)
	cmd.Stderr = os.Stderr

	return errors.Wrap(cmd.Run(), "running psql")
}

// Unpack takes the backed up contents and puts then imto a single
// SQL file
func (Engine) Unpack(r io.ReaderAt, size int64, destDir string) error {
	f, err := os.Create(path.Join(destDir, "backup.sql")) //#nosec:G304 // It's intended to write to use specified location
	if err != nil {
		return errors.Wrap(err, "creating output file")
	}

	if _, err = io.Copy(f, io.NewSectionReader(r, 0, size)); err != nil {
		return errors.Wrap(err, "copying file contents")
	}

	return errors.Wrap(f.Close(), "closing output file")
}

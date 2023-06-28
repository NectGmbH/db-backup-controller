// Package cockroach contains the implementation of the backupengine
// for CockroachDB
package cockroach

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	_ "github.com/Kount/pq-timeouts" // Required for CRDB connection
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"

	backupControllerV1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
	"github.com/NectGmbH/db-backup-controller/pkg/backupengine/base"
	"github.com/NectGmbH/db-backup-controller/pkg/backupengine/opts"
	"github.com/NectGmbH/db-backup-controller/pkg/storage/helper"
)

const (
	crdbCertDir = "/cockroach-certs"
	crdbTimeout = 10 // 10 Sseconds

	fileModeCert = 0o600
)

type (
	// Engine implements backupengine interface
	Engine struct {
		hdl http.Handler

		baseEngine base.Engine
		baseURL    string
		spec       backupControllerV1.DatabaseBackupSpec
	}
)

// New creates a new Engine instance
func New() *Engine { return &Engine{} }

// CreateBackup is used to instruct the backup engine to create
// a backup. The means of doing so depends on the engine itself.
func (e *Engine) CreateBackup(w io.Writer) error {
	if e.hdl != nil {
		// A listener is there, backup might be in progress
		return errors.New("backup listener still active")
	}

	u, err := url.Parse(e.baseURL)
	if err != nil {
		return errors.Wrap(err, "parsing base URL")
	}
	u.Path = "/crdb-backup"

	bw := newBackupWriter(w, logrus.NewEntry(logrus.StandardLogger()))
	defer func() {
		if err := bw.Close(); err != nil {
			logrus.WithError(err).Error("closing crdb backup-writer")
		}
	}()

	e.hdl = bw
	defer func() { e.hdl = nil }()

	db, err := e.crdbConnect()
	if err != nil {
		return errors.Wrap(err, "connecting to crdb")
	}
	defer func() {
		if err := db.Close(); err != nil {
			logrus.WithError(err).Error("closing crdb connection")
		}
	}()

	// This is fine-ish, we validate the name not to contain bad stuff
	if _, err = db.Exec(fmt.Sprintf("BACKUP DATABASE %s TO $1", e.spec.Cockroach.Database),
		u.String(),
	); err != nil {
		return errors.Wrap(err, "executing backup")
	}

	return nil
}

// GetPodSpec generates a pod-spec from the given backup
// specificiation containing required volume mounts from secrets
// or envFrom definitions (and possible other special cases).
// The PVC to store the backup is added later and must not be
// included in this spec
func (e *Engine) GetPodSpec(imagePrefix string) (coreV1.PodSpec, error) {
	podSpec, err := e.baseEngine.GetPodSpec()
	if err != nil {
		return podSpec, errors.Wrap(err, "getting base spec")
	}

	// Add certificate request
	podSpec.Volumes = append(podSpec.Volumes, coreV1.Volume{
		Name: "client-certs",
		VolumeSource: coreV1.VolumeSource{
			EmptyDir: &coreV1.EmptyDirVolumeSource{},
		},
	})

	// Add volume mount for certs
	podSpec.Containers[0].VolumeMounts = append(
		podSpec.Containers[0].VolumeMounts,
		coreV1.VolumeMount{Name: "client-certs", MountPath: crdbCertDir},
	)

	// Set cockroach image - we're not depending on external tools so
	// we don't care for versioning here and just use the default
	// version of the image
	podSpec.Containers[0].Image = strings.Join([]string{imagePrefix, "cockroach"}, "-")

	return podSpec, nil
}

// Init is called once per backup engine and allows to execute
// one-shot initialization tasks like registering new HTTP
// handlers
func (e *Engine) Init(options opts.InitOpts) error {
	if err := e.baseEngine.Init(options); err != nil {
		return errors.Wrap(err, "initializing base engine")
	}

	e.baseURL = options.BaseURL
	e.spec = options.Spec

	if err := e.validateDatabaseName(e.spec.Cockroach.Database); err != nil {
		return errors.Wrap(err, "validating database name")
	}

	if options.Mux != nil {
		options.Mux.PathPrefix("/crdb-backup").HandlerFunc(e.handleCRDBCommunication)
	}

	return nil
}

// RestoreBackup receives an io.Reader with the contents of the
// backup to be restored. The means of doing so depends on the
// engine itself. The contents of the reader will be the same
// the engine provided during the CreateBackup result
func (e *Engine) RestoreBackup(r helper.ReaderAtCloser, size int64) error {
	if e.hdl != nil {
		// A listener is there, restore might be in progress
		return errors.New("backup sender still active")
	}

	u, err := url.Parse(e.baseURL)
	if err != nil {
		return errors.Wrap(err, "parsing base URL")
	}
	u.Path = "/crdb-backup"

	br, err := newBackupReader(r, size, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		return errors.Wrap(err, "creating backup reader")
	}

	e.hdl = br
	defer func() { e.hdl = nil }()

	db, err := e.crdbConnect()
	if err != nil {
		return errors.Wrap(err, "connecting to crdb")
	}
	defer func() {
		if err := db.Close(); err != nil {
			logrus.WithError(err).Error("closing crdb connection")
		}
	}()

	// This is fine-ish, we validate the name not to contain bad stuff
	if _, err = db.Exec(fmt.Sprintf("RESTORE DATABASE %s FROM $1", e.spec.Cockroach.Database),
		u.String(),
	); err != nil {
		return errors.Wrap(err, "starting restore")
	}

	return nil
}

func (e Engine) crdbConnect() (*sql.DB, error) {
	certDir := crdbCertDir
	if v := os.Getenv("OVERRIDE_CRDB_CERT_DIR"); v != "" {
		certDir = v
	}

	if err := e.writeCertificates(certDir); err != nil {
		return nil, errors.Wrap(err, "writing certificates")
	}

	caCertPath := path.Join(certDir, "ca.crt")
	if e.spec.Cockroach.CertCAFromCluster {
		caCertPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	}

	db, err := sql.Open(
		"pq-timeouts",
		fmt.Sprintf(strings.Join([]string{
			"user=%s",
			"host=%s",
			"port=%d",
			"dbname=%s",
			"connect_timeout=%d",
			"sslrootcert=%s",
			"sslkey=%s",
			"sslcert=%s",
		}, " "),
			e.spec.Cockroach.User,
			e.spec.Cockroach.Host,
			e.spec.Cockroach.Port,
			e.spec.Cockroach.Database,
			crdbTimeout,
			caCertPath,
			path.Join(certDir, "client.key"),
			path.Join(certDir, "client.crt"),
		),
	)

	return db, errors.Wrap(err, "connecting to database")
}

func (e *Engine) handleCRDBCommunication(w http.ResponseWriter, r *http.Request) {
	if e.hdl == nil {
		// There is no handler: we did not want to communicate
		http.Error(w, "comms not requested", http.StatusNotFound)
		return
	}

	e.hdl.ServeHTTP(w, r)
}

func (*Engine) validateDatabaseName(dbName string) error {
	// In our SQL grammar, all values that accept an identifier must:
	// - Begin with a Unicode letter or an underscore (_). Subsequent
	//   characters can be letters, underscores, digits (0-9), or
	//   dollar signs ($).
	// - Not equal any SQL keyword unless the keyword is accepted by
	//   the element's syntax. For example, name accepts Unreserved or
	//   Column Name keywords.
	//
	// To bypass either of these rules, simply surround the identifier
	// with double-quotes ("). You can also use double-quotes to
	// preserve case-sensitivity in database, table, view, and column
	// names. However, all references to such identifiers must also
	// include double-quotes.
	//
	// https://www.cockroachlabs.com/docs/stable/keywords-and-identifiers.html

	if !regexp.MustCompile(`^[a-zA-Z0-9_$]+$`).MatchString(dbName) {
		return errors.New("invalid characters in database name")
	}

	return nil
}

func (e Engine) writeCertificates(baseDir string) error {
	for fn, content := range map[string]string{
		"ca.crt":     e.spec.Cockroach.CertCA.Value,
		"client.key": e.spec.Cockroach.CertKey.Value,
		"client.crt": e.spec.Cockroach.Cert.Value,
	} {
		if err := os.WriteFile(path.Join(baseDir, fn), []byte(content), fileModeCert); err != nil {
			return errors.Wrapf(err, "writing %s", fn)
		}
	}

	return nil
}

// Package opts contains helpers for the backup-engines to prevent
// import-cycles when importing engines into a registry
package opts

import (
	"github.com/gorilla/mux"

	backupControllerV1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
)

type (
	// InitOpts is a shared struct to configure a backup-engine
	InitOpts struct {
		// BaseURL specifies the URL the Mux is available at
		BaseURL string
		// Mux allows to register additional listener on for the HTTP
		// server afterwards available at the BaseURL
		Mux *mux.Router
		// Spec contains the configuration spec for the backup and
		// should be saved for later use for example in GetPodSpec
		Spec backupControllerV1.DatabaseBackupSpec
	}
)

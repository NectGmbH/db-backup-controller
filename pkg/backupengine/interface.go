// Package backupengine contains a definition how to implement
// a backup engine
package backupengine

import (
	"io"

	coreV1 "k8s.io/api/core/v1"

	"github.com/NectGmbH/db-backup-controller/pkg/backupengine/opts"
)

type (
	// Implementation defines how a backupengine implementation should look
	Implementation interface {
		// CreateBackup is used to instruct the backup engine to create
		// a backup. The means of doing so depends on the engine itself.
		CreateBackup(io.Writer) error
		// GetPodSpec generates a pod-spec from the given backup
		// specificiation containing required volume mounts from secrets
		// or envFrom definitions (and possible other special cases).
		// The mounted secret will be added by the controller.
		GetPodSpec(imagePrefix string) (coreV1.PodSpec, error)
		// Init is called once per backup engine and allows to execute
		// one-shot initialization tasks like registering new HTTP
		// handlers
		Init(opts.InitOpts) error
		// RestoreBackup receives an io.ReaderAt with the contents of
		// the backup to be restored and the size of the backup. The
		// means of doing so depends on the engine itself. The contents
		// of the reader will be the same the engine provided during
		// the CreateBackup result
		RestoreBackup(io.ReaderAt, int64) error
	}
)

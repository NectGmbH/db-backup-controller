// Package base contains a simple engine to return a predefined (YAML)
// pod-spec in order not to invent the wheel over and over again
package base

import (
	_ "embed"
	"io"

	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/NectGmbH/db-backup-controller/pkg/backupengine/opts"
	"github.com/NectGmbH/db-backup-controller/pkg/storage/helper"
)

//go:embed podSpec.yaml
var podSpecYAML []byte

type (
	// Engine is a baseic implementation of the backupengine interface
	Engine struct{}
)

// CreateBackup is not supported by this engine
func (Engine) CreateBackup(io.Writer) error {
	return errors.New("base-engine does not support backups")
}

// GetPodSpec generates a pod-spec from the given backup
// specificiation containing required volume mounts from secrets
// or envFrom definitions (and possible other special cases).
// The PVC to store the backup is added later and must not be
// included in this spec
func (Engine) GetPodSpec() (out coreV1.PodSpec, err error) {
	if err = yaml.UnmarshalStrict(podSpecYAML, &out); err != nil {
		return out, errors.Wrap(err, "unmarshalling base pod-spec")
	}

	return out, nil
}

// Init not used by this engine
func (Engine) Init(opts.InitOpts) error { return nil }

// RestoreBackup is not supported by this engine
func (Engine) RestoreBackup(helper.ReaderAtCloser) error {
	return errors.New("base-engine does not support backups")
}

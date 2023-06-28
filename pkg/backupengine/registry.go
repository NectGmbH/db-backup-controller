package backupengine

import "github.com/NectGmbH/db-backup-controller/pkg/backupengine/cockroach"

// GetByName contains a mapping of names to be specified in the
// backup spec to their Implementation
func GetByName(name string) Implementation { //nolint:ireturn // Returning the interface is intended here, this is a registry
	switch name {
	case "cockroach":
		return cockroach.New()

	default:
		return nil
	}
}

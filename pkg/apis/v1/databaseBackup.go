package v1

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

// FetchSecrets iterates through all Secret resources inside the
// spec and pulls their values from the original locations into the
// local instance
func (d *DatabaseBackupSpec) FetchSecrets(ctx context.Context, client kubernetes.Interface, namespace string) error {
	return fetchSecretsRecurse(ctx, d, client, namespace)
}

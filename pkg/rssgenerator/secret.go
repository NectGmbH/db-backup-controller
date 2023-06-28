package rssgenerator

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	v1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
)

func generateSecret(o Opts, res *Result) (err error) {
	sc, err := o.ControllerClient.BackupV1().
		DatabaseBackupStorageClasses().
		Get(context.TODO(), o.Backup.Spec.BackupStorageClass, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "fetching DatabaseBackupStorageClass")
	}

	if err = o.Backup.Spec.FetchSecrets(context.TODO(), o.K8sClient, o.Backup.Namespace); err != nil {
		return errors.Wrap(err, "fetching secret values for backup")
	}

	if err = sc.Spec.FetchSecrets(context.TODO(), o.K8sClient, o.TargetNamespace); err != nil {
		return errors.Wrap(err, "fetching secret values for storage class")
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    objectMetaLabelsFromDatabaseBackup(o.Backup),
			Name:      o.ResourceName,
			Namespace: o.TargetNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	// Nuke the status and resource version as it would create constant update cycles
	bCopy := o.Backup.DeepCopy()
	bCopy.ResourceVersion = ""
	bCopy.Status = v1.DatabaseBackupStatus{}

	if secret.Data["backup.yaml"], err = yaml.Marshal(bCopy); err != nil {
		return errors.Wrap(err, "marshalling DatabaseBackup")
	}

	if secret.Data["storageSpec.yaml"], err = yaml.Marshal(sc.Spec); err != nil {
		return errors.Wrap(err, "marshalling DatabaseBackupStorageClassSpec")
	}

	res.Secret = secret

	return nil
}

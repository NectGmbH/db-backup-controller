// Package rssgenerator contains generation of resources required to
// start a db-backup runner through the Kubernetes API
package rssgenerator

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	v1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
	"github.com/NectGmbH/db-backup-controller/pkg/generated/clientset/versioned"
)

type (
	// Opts contains the options given to any generator function
	Opts struct {
		Backup           *v1.DatabaseBackup
		ControllerClient versioned.Interface
		TargetNamespace  string
		ImagePrefix      string
		K8sClient        kubernetes.Interface
		LogLevel         string
		ResourceName     string
	}

	// Result contains the result from the generator function
	Result struct {
		Secret  *corev1.Secret
		Service *corev1.Service
		STS     *appsv1.StatefulSet

		Hash string `hash:"-"`
	}

	generator struct {
		name string
		fn   generatorFn
	}
	generatorFn func(Opts, *Result) error
)

// FromKubernetesResources generates the required resources to start
// a database backup runner
func FromKubernetesResources(o Opts) (res Result, err error) {
	for _, g := range []generator{
		{name: "secret", fn: generateSecret},
		{name: "service", fn: generateService},
		// STS accesses the secret, keep last
		{name: "sts", fn: generateSTS},
	} {
		if err = g.fn(o, &res); err != nil {
			return res, errors.Wrapf(err, "generating %s", g.name)
		}
	}

	// Calculate hash of generated resources
	if err = res.generateHash(); err != nil {
		return res, errors.Wrap(err, "generating hash")
	}

	return res, nil
}

func (r *Result) generateHash() error {
	rssH := sha256.New()
	nh, err := hashstructure.Hash(r, hashstructure.FormatV2, nil)
	if err != nil {
		return errors.Wrap(err, "hashing result")
	}
	if _, err = fmt.Fprintf(rssH, "%d", nh); err != nil {
		return errors.Wrap(err, "writing checksum to hash")
	}

	r.Hash = fmt.Sprintf("sha256:%x", rssH.Sum(nil))
	return nil
}

func objectMetaLabelsFromDatabaseBackup(b *v1.DatabaseBackup) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":           "db-backup",
		"app.kubernetes.io/instance":       strings.Join([]string{b.Namespace, b.Name}, "-"),
		"app.kubernetes.io/component":      "runner",
		"app.kubernetes.io/managed-by":     "db-backup-controller",
		"db-backup.nect.com/src-namespace": b.Namespace,
		"db-backup.nect.com/src-name":      b.Name,
	}
}

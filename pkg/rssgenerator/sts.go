package rssgenerator

import (
	"fmt"
	"strconv"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/NectGmbH/db-backup-controller/pkg/backupengine"
	"github.com/NectGmbH/db-backup-controller/pkg/backupengine/opts"
)

func generateSTS(o Opts, res *Result) (err error) {
	engine := backupengine.GetByName(o.Backup.Spec.DatabaseType)
	if engine == nil {
		return errors.New("unknown database type specified")
	}

	if err = engine.Init(opts.InitOpts{
		Spec: o.Backup.Spec,
	}); err != nil {
		return errors.Wrap(err, "initializing backup engine")
	}

	cfgHash, err := hashstructure.Hash(res.Secret, hashstructure.FormatV2, nil)
	if err != nil {
		return errors.Wrap(err, "hashing secret")
	}

	res.STS = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    objectMetaLabelsFromDatabaseBackup(o.Backup),
			Name:      o.ResourceName,
			Namespace: o.TargetNamespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: func(v int32) *int32 { return &v }(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: objectMetaLabelsFromDatabaseBackup(o.Backup),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"db-backup.nect.com/confighash": strconv.FormatUint(cfgHash, 10),
					},
					Labels: objectMetaLabelsFromDatabaseBackup(o.Backup),
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					MaxUnavailable: func(v intstr.IntOrString) *intstr.IntOrString { return &v }(intstr.FromInt(1)),
				},
			},
		},
	}

	if res.STS.Spec.Template.Spec, err = engine.GetPodSpec(o.ImagePrefix); err != nil {
		return errors.Wrap(err, "getting pod spec")
	}

	res.STS.Spec.Template.Spec.Volumes = append(res.STS.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "runner-config",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: res.Secret.Name,
				Optional:   func(v bool) *bool { return &v }(false),
			},
		},
	})

	for i := range res.STS.Spec.Template.Spec.Containers {
		res.STS.Spec.Template.Spec.Containers[i].Env = append(
			res.STS.Spec.Template.Spec.Containers[i].Env,
			corev1.EnvVar{Name: "BASE_URL", Value: fmt.Sprintf("http://%s.%s.svc.cluster.local:3000/", o.ResourceName, o.TargetNamespace)},
			corev1.EnvVar{Name: "LOG_LEVEL", Value: o.LogLevel},
		)
	}

	return nil
}

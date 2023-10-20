package rssgenerator

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	servicePort       = 3000
	serviceNameMaxLen = 63
)

func generateService(o Opts, res *Result) (err error) {
	res.Service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    objectMetaLabelsFromDatabaseBackup(o.Backup),
			Name:      o.ResourceName,
			Namespace: o.TargetNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "api",
					Protocol: corev1.ProtocolTCP,
					Port:     servicePort,
				},
			},
			Selector: objectMetaLabelsFromDatabaseBackup(o.Backup),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}

	return nil
}

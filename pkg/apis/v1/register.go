package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// GroupName defines the group name of this API package
	GroupName = "backup.nect.com"
	// GroupVersion defines the current group version of this package
	GroupVersion = "v1"
)

// SchemeGroupVersion contains a schema GroupVersion of the exported
// GroupName and GroupVersion in this api package
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

var (
	// SchemeBuilder contains a runtime.SchemeBuilder with types
	// available in this api package
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme contains the AddToScheme function of the SchemeBuilder
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&DatabaseBackup{},
		&DatabaseBackupList{},
		&DatabaseBackupStorageClass{},
		&DatabaseBackupStorageClassList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

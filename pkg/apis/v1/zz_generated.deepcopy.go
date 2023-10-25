//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1

import (
	labelmanager "github.com/NectGmbH/db-backup-controller/pkg/labelmanager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CockroachConfig) DeepCopyInto(out *CockroachConfig) {
	*out = *in
	out.Cert = in.Cert
	out.CertCA = in.CertCA
	out.CertKey = in.CertKey
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CockroachConfig.
func (in *CockroachConfig) DeepCopy() *CockroachConfig {
	if in == nil {
		return nil
	}
	out := new(CockroachConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseBackup) DeepCopyInto(out *DatabaseBackup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseBackup.
func (in *DatabaseBackup) DeepCopy() *DatabaseBackup {
	if in == nil {
		return nil
	}
	out := new(DatabaseBackup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DatabaseBackup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseBackupList) DeepCopyInto(out *DatabaseBackupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DatabaseBackup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseBackupList.
func (in *DatabaseBackupList) DeepCopy() *DatabaseBackupList {
	if in == nil {
		return nil
	}
	out := new(DatabaseBackupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DatabaseBackupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseBackupSpec) DeepCopyInto(out *DatabaseBackupSpec) {
	*out = *in
	if in.RetentionConfig != nil {
		in, out := &in.RetentionConfig, &out.RetentionConfig
		*out = make(labelmanager.RetentionConfig, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Cockroach != nil {
		in, out := &in.Cockroach, &out.Cockroach
		*out = new(CockroachConfig)
		**out = **in
	}
	if in.MySQL != nil {
		in, out := &in.MySQL, &out.MySQL
		*out = new(MySQLConfig)
		**out = **in
	}
	if in.Postgres != nil {
		in, out := &in.Postgres, &out.Postgres
		*out = new(PostgresConfig)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseBackupSpec.
func (in *DatabaseBackupSpec) DeepCopy() *DatabaseBackupSpec {
	if in == nil {
		return nil
	}
	out := new(DatabaseBackupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseBackupStatus) DeepCopyInto(out *DatabaseBackupStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseBackupStatus.
func (in *DatabaseBackupStatus) DeepCopy() *DatabaseBackupStatus {
	if in == nil {
		return nil
	}
	out := new(DatabaseBackupStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseBackupStorageClass) DeepCopyInto(out *DatabaseBackupStorageClass) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseBackupStorageClass.
func (in *DatabaseBackupStorageClass) DeepCopy() *DatabaseBackupStorageClass {
	if in == nil {
		return nil
	}
	out := new(DatabaseBackupStorageClass)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DatabaseBackupStorageClass) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseBackupStorageClassList) DeepCopyInto(out *DatabaseBackupStorageClassList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DatabaseBackupStorageClass, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseBackupStorageClassList.
func (in *DatabaseBackupStorageClassList) DeepCopy() *DatabaseBackupStorageClassList {
	if in == nil {
		return nil
	}
	out := new(DatabaseBackupStorageClassList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DatabaseBackupStorageClassList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseBackupStorageClassSpec) DeepCopyInto(out *DatabaseBackupStorageClassSpec) {
	*out = *in
	if in.BackupLocations != nil {
		in, out := &in.BackupLocations, &out.BackupLocations
		*out = make([]DatabaseBackupStorageLocation, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseBackupStorageClassSpec.
func (in *DatabaseBackupStorageClassSpec) DeepCopy() *DatabaseBackupStorageClassSpec {
	if in == nil {
		return nil
	}
	out := new(DatabaseBackupStorageClassSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseBackupStorageLocation) DeepCopyInto(out *DatabaseBackupStorageLocation) {
	*out = *in
	out.StorageAccessKeyID = in.StorageAccessKeyID
	out.StorageSecretAccessKey = in.StorageSecretAccessKey
	out.EncryptionPass = in.EncryptionPass
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseBackupStorageLocation.
func (in *DatabaseBackupStorageLocation) DeepCopy() *DatabaseBackupStorageLocation {
	if in == nil {
		return nil
	}
	out := new(DatabaseBackupStorageLocation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MySQLConfig) DeepCopyInto(out *MySQLConfig) {
	*out = *in
	out.Pass = in.Pass
	out.User = in.User
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MySQLConfig.
func (in *MySQLConfig) DeepCopy() *MySQLConfig {
	if in == nil {
		return nil
	}
	out := new(MySQLConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresConfig) DeepCopyInto(out *PostgresConfig) {
	*out = *in
	out.Pass = in.Pass
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresConfig.
func (in *PostgresConfig) DeepCopy() *PostgresConfig {
	if in == nil {
		return nil
	}
	out := new(PostgresConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Secret) DeepCopyInto(out *Secret) {
	*out = *in
	out.FromSecret = in.FromSecret
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Secret.
func (in *Secret) DeepCopy() *Secret {
	if in == nil {
		return nil
	}
	out := new(Secret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretKeyRef) DeepCopyInto(out *SecretKeyRef) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretKeyRef.
func (in *SecretKeyRef) DeepCopy() *SecretKeyRef {
	if in == nil {
		return nil
	}
	out := new(SecretKeyRef)
	in.DeepCopyInto(out)
	return out
}

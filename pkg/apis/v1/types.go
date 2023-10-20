package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	labelmanager "github.com/NectGmbH/db-backup-controller/pkg/labelmanager"
)

// DatabaseBackup contains the Kubernetes document for the
// DatabaseBackupSpec
//
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
type DatabaseBackup struct {
	metav1.TypeMeta   `json:",inline"` //revive:disable-line:struct-tag // "inline" is valid
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseBackupSpec   `json:"spec"`
	Status DatabaseBackupStatus `json:"status,omitempty"`
}

// DatabaseBackupList contains a list of DatabaseBackup
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DatabaseBackupList struct {
	metav1.TypeMeta `json:",inline"` //revive:disable-line:struct-tag // "inline" is valid
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DatabaseBackup `json:"items"`
}

// DatabaseBackupSpec describes how the backup of the database
// should look and how to connect to the database
//
// +kubebuilder:object:generate=true
type DatabaseBackupSpec struct {
	// DatabaseType specifies the type of the database to be backed up.
	// This changes the behaviour of the backup engine used in the
	// background and needs to match the provisioned database.
	//
	// +kubebuilder:validation:Enum={cockroach, mysql, postgres}
	DatabaseType string `json:"databaseType"`

	// DatabaseVersion is an arbitrary string the database driver uses
	// to determine how to handle the exact version of the database.
	// This for example is used to handle the different approaches used
	// by Cockroach v21 and v23.
	//
	// +kubebuilder:validation:Optional
	DatabaseVersion string `json:"databaseVersion"`

	// BackupStorageClass defines where to store the backups (must
	// exist before)
	BackupStorageClass string `json:"backupStorageClass"`

	// BackupCron specifies a cron entry (5 fields) when to execute
	// the backup runs. If set to other than empty string will
	// override the BackupInterval
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`((((\d+,)+\d+|(\d+(\/|-)\d+)|\d+|\*) ?){5})`
	BackupCron string `json:"backupCron"`
	// BackupIntervalHours is a simpler method of specifying the
	// backup schedule: The process will wait for the current time
	// to be divisible by the given interval. (i.e. if you specify
	// 2h the backup will run 00:00, 02:00, 04:00, ...)
	//
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Optional
	BackupIntervalHours int64 `json:"backupInterval"`
	// RetentionConfig defines the retention rules applied to store
	// old copies of the backup in case generation-principle backups
	// are enabled. If left to nil the default retention config is
	// used
	//
	// +kubebuilder:validation:Optional
	RetentionConfig labelmanager.RetentionConfig `json:"retentionConfig"`
	// UseSingleBackupTarget defines whether to upload in the same
	// place all the time (S3 server then needs to take care of
	// rotating / revisioning backup file and backup can only restore
	// this single instance, not to a point in time) or whether to
	// use generation principle backups and treat the storage target
	// as a prefix.
	//
	// +kubebuilder:default=false
	// +kubebuilder:validation:Optional
	UseSingleBackupTarget bool `json:"useSingleBackupTarget"`

	// Engine config

	// Cockroach defines the required values for a Cockroach backup
	//
	// +kubebuilder:validation:Optional
	Cockroach *CockroachConfig `json:"cockroach,omitempty"`
	// MySQL defines the required values for a MySQL backup
	//
	// +kubebuilder:validation:Optional
	MySQL *MySQLConfig `json:"mysql,omitempty"`
	// Postgres defines the required values for a PostgreSQL backup
	//
	// +kubebuilder:validation:Optional
	Postgres *PostgresConfig `json:"postgres,omitempty"`
}

// DatabaseBackupStatus represents a status of a DatabaseBackup resource
type DatabaseBackupStatus struct {
	// Collection of conditions
	Conditions []metav1.Condition `json:"conditions"`
	// Last applied hash of the created resources for comparison of a
	// generated update
	//
	// +kubebuilder:validation:Optional
	Hash string `json:"hash,omitempty"`
}

// CockroachConfig contains the values required for the
// backup-engine to backup a single database on a Cockroach
// server
//
// +kubebuilder:object:generate=true
type CockroachConfig struct {
	// Database specifies the database to be backed up
	Database string `json:"database"`
	// Host specifies the IP or DNS name to connect to
	Host string `json:"host"`
	// Port specifies the port the database is listening on (usually 3306)
	Port int64 `json:"port"`
	// User specifies the user to use for connection
	User string `json:"user"`

	// Cert is the client certificate used to authenticate against CRDB
	Cert Secret `json:"cert"`
	// TLSCertCA specifies the CA certificate and should be specified
	// as reference to a key in a secret if CA is managed as a secret
	//
	// +kubebuilder:validation:Optional
	CertCA Secret `json:"certCA"`
	// CertCAFromCluster specifies the cluster certificate should be
	// used instead of the certCA provided certificate. If this is set
	// to true the certCA key will be ignored.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	CertCAFromCluster bool `json:"certCAFromCluster"`
	// CertKey is the private key for the given client certificate
	CertKey Secret `json:"certKey"`
}

// MySQLConfig contains the values required for the backup-engine
// to backup a single database on a MySQL server
//
// +kubebuilder:object:generate=true
type MySQLConfig struct {
	// Database specifies the database to be backed up
	Database string `json:"database"`
	// Host specifies the IP or DNS name to connect to
	Host string `json:"host"`
	// Pass specifies a reference to or the value of the users password
	Pass Secret `json:"pass"`
	// Port specifies the port the database is listening on (usually 3306)
	Port int64 `json:"port"`
	// User specifies the user or a reference to it to use for connection
	User Secret `json:"user"`
}

// PostgresConfig contains the values required for the
// backup-engine to backup a single database on a Postgres
// server
//
// +kubebuilder:object:generate=true
type PostgresConfig struct {
	// Database specifies the database to be backed up
	Database string `json:"database"`
	// Host specifies the IP or DNS name to connect to
	Host string `json:"host"`
	// Port specifies the port the database is listening on (usually 3306)
	Port int64 `json:"port"`
	// User specifies the user or a reference to it to use for connection
	User string `json:"user"`
	// Pass specifies a reference to or the value of the users password
	Pass Secret `json:"pass"`
	// DSNParameters is an optional map[string]string to define further
	// parameters for the DSN like `sslmode`, `connect_timeout`, etc.
	// The `user`, `host`, `port`, `dbname`, `pass` parameters are NOT
	// settable through this and are overwritten unconditionally.
	DSNParameters map[string]string `json:"dsnParameters"`
}

// DatabaseBackupStorageClass contains the Kubernetes document for
// the DatabaseBackupStorageClassSpec
//
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
type DatabaseBackupStorageClass struct {
	metav1.TypeMeta   `json:",inline"` //revive:disable-line:struct-tag // "inline" is valid
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DatabaseBackupStorageClassSpec `json:"spec"`
}

// DatabaseBackupStorageClassList contains a list of
// DatabaseBackupStorageClass
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
type DatabaseBackupStorageClassList struct {
	metav1.TypeMeta `json:",inline"` //revive:disable-line:struct-tag // "inline" is valid
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DatabaseBackupStorageClass `json:"items"`
}

// DatabaseBackupStorageClassSpec contains a list of locations to
// write the backups to when using this DatabaseBackupStorageClass
//
// +kubebuilder:object:generate=true
type DatabaseBackupStorageClassSpec struct {
	// BackupLocations defines a number of locations the backups
	// written to this DatabaseBackupStorageClass should be uploaded
	// to.
	//
	// +kubebuilder:validation:MinItems=1
	BackupLocations []DatabaseBackupStorageLocation `json:"backupLocations"`
}

// DatabaseBackupStorageLocation describes how to connect to the
// storage location for uploading the backup
//
// +kubebuilder:object:generate=true
type DatabaseBackupStorageLocation struct {
	// StorageType defines which storage engine to load for this
	// storage location
	//
	// +kubebuilder:validation:Enum={s3}
	StorageType string `json:"storageType"`

	// StorageEndpoint defines the MinIO / S3 endpoint to connect to
	StorageEndpoint string `json:"storageEndpoint"`
	// StorageAccessKeyID and StorageSecretAccessKey define the
	// credentials to access the StorageBucket
	StorageAccessKeyID     Secret `json:"storageAccessKeyID"`
	StorageSecretAccessKey Secret `json:"storageSecretAccessKey"`
	// StorageBucket defines to which bucket to upload the files
	StorageBucket string `json:"storageBucket"`
	// StorageLocation defines the location the bucket exists in
	// (i.e. "minio", "eu-west-1", ...)
	StorageLocation string `json:"storageLocation"`
	// StorageUseSSL defines whether to use TLS encrypted connection
	// to storage
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	StorageUseSSL bool `json:"storageUseSSL"`
	// StorageInsecureSkipVerify defines whether to skip TLS cert
	// verify as of unknown CA / ...
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	StorageInsecureSkipVerify bool `json:"storageInsecureSkipVerify"`

	// EncryptionPass defines the passphrase to be used to encrypt the
	// backup before writing it to the storage server specified in this
	// location. Leaving this empty will DISABLE encryption!
	//
	// +kubebuilder:validation:Optional
	EncryptionPass Secret `json:"encryptionPass"`
}

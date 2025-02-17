// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	apisv1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
	versioned "github.com/NectGmbH/db-backup-controller/pkg/generated/clientset/versioned"
	internalinterfaces "github.com/NectGmbH/db-backup-controller/pkg/generated/informers/externalversions/internalinterfaces"
	v1 "github.com/NectGmbH/db-backup-controller/pkg/generated/listers/apis/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// DatabaseBackupInformer provides access to a shared informer and lister for
// DatabaseBackups.
type DatabaseBackupInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.DatabaseBackupLister
}

type databaseBackupInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewDatabaseBackupInformer constructs a new informer for DatabaseBackup type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewDatabaseBackupInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredDatabaseBackupInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredDatabaseBackupInformer constructs a new informer for DatabaseBackup type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredDatabaseBackupInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BackupV1().DatabaseBackups(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BackupV1().DatabaseBackups(namespace).Watch(context.TODO(), options)
			},
		},
		&apisv1.DatabaseBackup{},
		resyncPeriod,
		indexers,
	)
}

func (f *databaseBackupInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredDatabaseBackupInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *databaseBackupInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisv1.DatabaseBackup{}, f.defaultInformer)
}

func (f *databaseBackupInformer) Lister() v1.DatabaseBackupLister {
	return v1.NewDatabaseBackupLister(f.Informer().GetIndexer())
}

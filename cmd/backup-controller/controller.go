package main

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	v1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
	"github.com/NectGmbH/db-backup-controller/pkg/generated/clientset/versioned"
	"github.com/NectGmbH/db-backup-controller/pkg/generated/informers/externalversions"
)

type (
	controller struct {
		crdClient  versioned.Interface
		kubeClient kubernetes.Interface

		informerSyncs []cache.InformerSynced
		queue         workqueue.RateLimitingInterface
	}

	queueEntry struct {
		action   queueEntryAction
		key      string
		queuedAt time.Time
		reason   string
	}

	queueEntryAction uint8
)

const (
	queueEntryActionAdd queueEntryAction = iota
	queueEntryActionUpdate
	queueEntryActionDelete
)

func newController(
	crdClient versioned.Interface,
	kubeClient kubernetes.Interface,
) *controller {
	return &controller{
		crdClient:  crdClient,
		kubeClient: kubeClient,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "com.nect.db-backup"),
	}
}

// RegisterDatabaseBackupInformer registers the required event
// handlers to handle changes on databaseBackup CRDs
func (c *controller) RegisterDatabaseBackupInformer(factory externalversions.SharedInformerFactory) error {
	informer := factory.Backup().V1().DatabaseBackups().Informer()

	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) { c.enqueue(obj, queueEntryActionAdd, "databaseBackup added") },
		// We don't care about it being gone, we're acting on the update
		// marking it for deletion
		UpdateFunc: func(_, obj any) { c.enqueue(obj, queueEntryActionUpdate, "databaseBackup updated") },
	}); err != nil {
		return errors.Wrap(err, "adding event handlers")
	}

	logrus.Info("registered DatabaseBackups handlers")

	c.informerSyncs = append(c.informerSyncs, informer.HasSynced)
	return nil
}

// RegisterDatabaseBackupStorageClassInformer registers the required
// event handlers to handle changes on databaseBackupStorageClass CRDs
func (c *controller) RegisterDatabaseBackupStorageClassInformer(factory externalversions.SharedInformerFactory) error {
	informer := factory.Backup().V1().DatabaseBackupStorageClasses().Informer()

	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// We don't care about "Add", needs to have a reference
		DeleteFunc: func(obj any) { c.enqueue(obj, queueEntryActionUpdate, "databaseBackupStorageClass deleted") },
		UpdateFunc: func(_, obj any) { c.enqueue(obj, queueEntryActionUpdate, "databaseBackupStorageClass updated") },
	}); err != nil {
		return errors.Wrap(err, "adding event handlers")
	}

	logrus.Info("registered DatabaseBackupStorageClasses handlers")

	c.informerSyncs = append(c.informerSyncs, informer.HasSynced)
	return nil
}

// RegisterSecretInformer registers the required event handlers to
// handle changes on secret resources
func (c *controller) RegisterSecretInformer(factory informers.SharedInformerFactory) error {
	informer := factory.Core().V1().Secrets().Informer()

	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// We don't care about "Add", needs to have a reference
		DeleteFunc: func(obj any) { c.enqueue(obj, queueEntryActionUpdate, "secret deleted") },
		UpdateFunc: func(_, obj any) { c.enqueue(obj, queueEntryActionUpdate, "secret updated") },
	}); err != nil {
		return errors.Wrap(err, "adding event handlers")
	}

	logrus.Info("registered Secrets handlers")

	c.informerSyncs = append(c.informerSyncs, informer.HasSynced)
	return nil
}

// Run starts the main processing flow of the controller
func (c *controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	logrus.Info("starting controller run")
	defer logrus.Info("shutting down controller")

	if !cache.WaitForCacheSync(stopCh, c.informerSyncs...) {
		return
	}

	logrus.WithField("workers", workers).Info("starting controller workers")

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *controller) enqueue(obj any, action queueEntryAction, reason string) {
	var (
		key  string
		keys []string
		err  error
	)

	switch obj.(type) {
	case *v1.DatabaseBackup:
		if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
			// We don't log the object for security reasons: It might be a secret!
			utilruntime.HandleError(errors.Wrap(err, "getting key for object"))
			return
		}

		keys = []string{key}

	default:
		utilruntime.HandleError(errors.Errorf("received %T but not prepared to handle", obj))
	}

	for _, key = range keys {
		q := &queueEntry{action: action, key: key, queuedAt: time.Now(), reason: reason}
		q.Logger().Debug("enqueing databaseBackup for processing")
		c.queue.Add(q)
	}
}

func (c *controller) processNextWorkItem() bool {
	qei, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(qei)

	qe, ok := qei.(*queueEntry)
	if !ok {
		// However this happened: Queue did not contain valid entry
		logrus.WithError(errors.Errorf("queue entry had wrong type %T", qei)).Error("handling invalid queue entry")
		return true
	}

	var handlerFn func(*queueEntry) error
	switch qe.action {
	case queueEntryActionAdd:
		handlerFn = c.handleDatabaseBackupAdd

	case queueEntryActionUpdate:
		handlerFn = c.handleDatabaseBackupUpdate

	case queueEntryActionDelete:
		handlerFn = c.handleDatabaseBackupDelete
	}

	if err := handlerFn(qe); err != nil {
		if qe.queuedAt.After(time.Now().Add(-cfg.RescanInterval)) {
			qe.Logger().WithError(err).Error("handling queue entry")
			c.queue.AddRateLimited(qei)
		} else {
			qe.Logger().WithError(err).Error("handling queue entry, dropping after entry timeout")
		}

		return true
	}

	c.queue.Forget(qei)
	return true
}

func (c *controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (q queueEntry) Fetch(crdClient versioned.Interface) (*v1.DatabaseBackup, error) {
	ns, name, err := cache.SplitMetaNamespaceKey(q.key)
	if err != nil {
		return nil, errors.Wrap(err, "splitting namespace key")
	}

	b, err := crdClient.BackupV1().DatabaseBackups(ns).Get(context.Background(), name, metav1.GetOptions{})
	return b, errors.Wrap(err, "fetching backup object")
}

func (q queueEntry) Logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"age":    time.Since(q.queuedAt),
		"key":    q.key,
		"reason": q.reason,
	})
}

func (q queueEntry) String() string { return strings.Join([]string{q.reason, q.key}, ": ") }

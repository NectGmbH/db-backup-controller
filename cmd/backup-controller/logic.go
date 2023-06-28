package main

import (
	"context"
	"crypto/sha1" //#nosec:G505 // Only used to shorten names
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Luzifer/go_helpers/v2/str"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/NectGmbH/db-backup-controller/pkg/apis/v1"
	"github.com/NectGmbH/db-backup-controller/pkg/rssgenerator"
)

const finalizer = "db-backup.nect.com/finalizer"

type (
	k8sDeletable interface {
		Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	}
)

func (c controller) handleDatabaseBackupAdd(q *queueEntry) error {
	q.Logger().Debug("handling DatabaseBackup add")

	b, err := q.Fetch(c.crdClient)
	if err != nil {
		return err
	}

	for _, f := range b.Finalizers {
		if f == finalizer {
			// Our finalizer is set, we don't need two of them
			q.Logger().Debug("finalizer present, noop")
			return nil
		}
	}

	// We just stick a finalizer and the status to it and let it return
	// in the update loop we just triggered through adding the
	// finalizer to do the real magic. This way we make sure it exists,
	// we can patch it and it doesn't sneak away while we're preparing
	// stuff for it.
	q.Logger().Info("initializing DatabaseBackup")
	return errors.Wrap(c.claimBackupObject(b), "updating DatabaseBackup")
}

func (c controller) handleDatabaseBackupUpdate(q *queueEntry) error {
	q.Logger().Debug("handling DatabaseBackup update")

	b, err := q.Fetch(c.crdClient)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Well, that queue entry is gone, no need to work on it so
			// lets call it a day and successful processing!
			return nil
		}
		return err
	}

	if !str.StringInSlice(finalizer, b.Finalizers) {
		// The heck? We shouldn't have an update for something without
		// finalizer applied. There is something seriously wrong but
		// we take it and do an initialization first
		return c.handleDatabaseBackupAdd(q)
	}

	if b.DeletionTimestamp != nil {
		// It has a timestamp? Nice, then it is about to die. Lets ensure
		// we cleaned up before that happens!
		return c.handleDatabaseBackupDelete(q)
	}

	opts := rssgenerator.Opts{
		Backup:           b,
		ControllerClient: c.crdClient,
		TargetNamespace:  cfg.TargetNamespace,
		ImagePrefix:      cfg.ImagePrefix,
		K8sClient:        c.kubeClient,
		LogLevel:         q.Logger().Logger.GetLevel().String(),
	}

	if opts.ResourceName, err = c.deriveName(b.Namespace, b.Name); err != nil {
		return errors.Wrap(err, "generating resource name")
	}

	rss, err := rssgenerator.FromKubernetesResources(opts)
	if err != nil {
		return errors.Wrap(err, "generating resources")
	}

	if rss.Hash == b.Status.Hash {
		// We should be fine, the generated resources matches the
		// previous generated resources and we stored that hash. So if
		// nobody messed with it the state should be fine.
		//
		// NOTE(kahlers): This is not exactly state-enforcing and we
		// might need to reconsider this exit instead of just patching
		// our target resources. On the other hand this is like majority
		// of other controllers behave and those resources can only get
		// modified by global-admins with cluster-write-access. Therefore
		// ThisIsFine.gif
		q.Logger().Debug("hash matches, noop")
		return nil
	}

	status := b.Status
	defer func() {
		b.Status = status
		if err := c.updateStatus(b); err != nil {
			q.Logger().WithError(err).Error("updating status")
		}
	}()

	for statusKey, uf := range map[v1.DatabaseBackupStatusCondition]func(
		rss rssgenerator.Result,
		status *v1.DatabaseBackupStatus,
	) error{
		v1.ConditionSecretExists:  c.upsertSecret,
		v1.ConditionServiceExists: c.upsertService,
		v1.ConditionSTSExists:     c.upsertStatefulSet,
	} {
		if err = uf(rss, &status); err != nil {
			status.Set(statusKey, b.Generation, metav1.ConditionUnknown, "handleDatabaseBackupUpdate", "Update caused error")
			return errors.Wrap(err, "updating component")
		}

		status.Set(statusKey, b.Generation, metav1.ConditionTrue, "handleDatabaseBackupUpdate", "Updated successfully")
	}

	// Finally if everything went well we update the hash
	status.Hash = rss.Hash

	return nil
}

func (c controller) handleDatabaseBackupDelete(q *queueEntry) error {
	q.Logger().Debug("handling DatabaseBackup delete")

	b, err := q.Fetch(c.crdClient)
	if err != nil {
		return err
	}

	rssName, err := c.deriveName(b.Namespace, b.Name)
	if err != nil {
		return errors.Wrap(err, "generating resource name")
	}

	q.Logger().Info("removing related resources")

	for rssType, d := range map[string]k8sDeletable{
		"secret":  c.kubeClient.CoreV1().Secrets(cfg.TargetNamespace),
		"service": c.kubeClient.CoreV1().Services(cfg.TargetNamespace),
		"sts":     c.kubeClient.AppsV1().StatefulSets(cfg.TargetNamespace),
	} {
		if err = d.Delete(context.Background(), rssName, metav1.DeleteOptions{}); err != nil {
			if !k8sErrors.IsNotFound(err) {
				// NotFound error would be fine, this is none, so bail out
				return errors.Wrapf(err, "deleting %s", rssType)
			}

			q.Logger().WithField("rss_type", rssType).Warn("resource was not found")
		}
	}

	q.Logger().Info("removing finalizer")
	return errors.Wrap(c.removeFinalizer(b), "removing finalizer")
}

func (c controller) claimBackupObject(b *v1.DatabaseBackup) error {
	finalizers := []string{finalizer}

	for _, f := range b.Finalizers {
		if f == finalizer {
			continue
		}
		finalizers = append(finalizers, f)
	}

	payload, err := json.Marshal([]map[string]any{{
		"op":    "add",
		"path":  "/metadata/finalizers",
		"value": finalizers,
	}})
	if err != nil {
		return errors.Wrap(err, "marshalling patch payload")
	}

	if b, err = c.crdClient.BackupV1().
		DatabaseBackups(b.Namespace).
		Patch(context.Background(), b.Name, types.JSONPatchType, payload, metav1.PatchOptions{}); err != nil {
		return errors.Wrap(err, "patching backup object")
	}

	initStatus := v1.DatabaseBackupStatus{}
	initStatus.Init(b.Generation)
	b.Status = initStatus

	return errors.Wrap(c.updateStatus(b), "setting initial status")
}

func (controller) deriveName(namespace, name string) (string, error) {
	h := sha1.New() //#nosec:G401 // Only used for shortening names
	if _, err := h.Write([]byte(strings.Join([]string{"runner", namespace, name}, "-"))); err != nil {
		return "", errors.Wrap(err, "writing svc name to hash")
	}

	return fmt.Sprintf("sha1-%x", h.Sum(nil)), nil
}

func (c controller) removeFinalizer(b *v1.DatabaseBackup) error {
	var finalizers []string

	for _, f := range b.Finalizers {
		if f == finalizer {
			continue
		}
		finalizers = append(finalizers, f)
	}

	if len(finalizers) == len(b.Finalizers) {
		// We did not remove any, doesn't seem to us who prevent deletion
		return nil
	}

	payload, err := json.Marshal([]map[string]any{{
		"op":    "replace",
		"path":  "/metadata/finalizers",
		"value": finalizers,
	}})
	if err != nil {
		return errors.Wrap(err, "marshalling patch payload")
	}

	_, err = c.crdClient.BackupV1().
		DatabaseBackups(b.Namespace).
		Patch(context.Background(), b.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	return errors.Wrap(err, "patching backup object")
}

func (c controller) updateStatus(b *v1.DatabaseBackup) error {
	s := b.Status
	s.CalculateReady(b.Generation)
	b.Status = s

	_, err := c.crdClient.BackupV1().
		DatabaseBackups(b.Namespace).
		UpdateStatus(context.Background(), b, metav1.UpdateOptions{})
	return errors.Wrap(err, "updating status")
}

func (c controller) upsertSecret(rss rssgenerator.Result, _ *v1.DatabaseBackupStatus) (err error) {
	intf := c.kubeClient.CoreV1().Secrets(cfg.TargetNamespace)

	_, err = intf.Update(context.Background(), rss.Secret, metav1.UpdateOptions{})

	if k8sErrors.IsNotFound(err) {
		// It doesn't exist, lets create it
		_, err = intf.Create(context.Background(), rss.Secret, metav1.CreateOptions{})
	}

	return errors.Wrap(err, "upserting secret")
}

func (c controller) upsertService(rss rssgenerator.Result, _ *v1.DatabaseBackupStatus) (err error) {
	intf := c.kubeClient.CoreV1().Services(cfg.TargetNamespace)

	svcJSON, err := json.Marshal(rss.Service)
	if err != nil {
		return errors.Wrap(err, "marshalling service to JSON")
	}
	// NOTE(kahlers): Don't try an update here, you cannot use update
	// without previously fetching the object already present in the
	// cluster and patching it to apply the changes. The only way I
	// found to just generate the resource and force the state was to
	// do a merge-patch.
	_, err = intf.Patch(context.Background(), rss.Service.Name, types.MergePatchType, svcJSON, metav1.PatchOptions{})

	if k8sErrors.IsNotFound(err) {
		// It doesn't exist, lets create it
		_, err = intf.Create(context.Background(), rss.Service, metav1.CreateOptions{})
	}

	return errors.Wrap(err, "upserting service")
}

func (c controller) upsertStatefulSet(rss rssgenerator.Result, _ *v1.DatabaseBackupStatus) (err error) {
	intf := c.kubeClient.AppsV1().StatefulSets(cfg.TargetNamespace)

	stsJSON, err := json.Marshal(rss.STS)
	if err != nil {
		return errors.Wrap(err, "marshalling sts to JSON")
	}
	// NOTE(kahlers): Don't try an update here, you cannot use update
	// without previously fetching the object already present in the
	// cluster and patching it to apply the changes. The only way I
	// found to just generate the resource and force the state was to
	// do a merge-patch.
	_, err = intf.Patch(context.Background(), rss.STS.Name, types.MergePatchType, stsJSON, metav1.PatchOptions{})

	if k8sErrors.IsNotFound(err) {
		// It doesn't exist, lets create it
		_, err = intf.Create(context.Background(), rss.STS, metav1.CreateOptions{})
	}

	return errors.Wrap(err, "upserting sts")
}

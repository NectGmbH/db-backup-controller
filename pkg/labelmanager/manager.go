// Package labelmanager contains a store to manage retention of
// backups through "labels" assigned in a specific format
package labelmanager

import (
	"io"
	"time"

	"github.com/itchyny/timefmt-go"
	"github.com/pkg/errors"
)

type (
	// Manager is a helper to organize and book-keep the entries based
	// on a Grandfather-Father-Son principle
	Manager struct {
		retention RetentionConfig

		store *retentionStore
	}
)

var (
	// ErrNoLabelsAdded signalizes when adding an entry all possible
	// labels were already present and the entry has not been added
	ErrNoLabelsAdded = errors.New("no labels were available to add")
	// ErrNoBackupFound signalizes there was no backup older than the
	// given point-in-time
	ErrNoBackupFound = errors.New("no backup found for point-in-time")
)

// New creates a new Manager configured with the baseDir and the
// RetentionConfig defining how long to keep backups
//
// if labelStorage is nil, an empty manager will be initialized
// if retention is nil, the DefaultRetentionConfig will be used
func New(labelStorage io.Reader, retention RetentionConfig) (*Manager, error) {
	store := newRetentionStore()
	if labelStorage != nil {
		if err := store.Load(labelStorage); err != nil {
			return nil, errors.Wrap(err, "loading retention store")
		}
	}

	if retention == nil {
		retention = DefaultRetentionConfig
	}

	return &Manager{
		retention: retention,

		store: store,
	}, nil
}

// Add adds a new backup to the manager. When adding it is assigned
// labels defined by the RetentionConfig in case they are not already
// assigned to any other backup. This ensures the generations are
// kept for as long as the RetentionConfig defines
//
// Returns ErrNoLabelsAdded in case all possible labels were already
// set. In this case the entry is not added to the Manager / store.
func (m Manager) Add(entryName string) error {
	var addedLabels int

	for format, retainFor := range m.retention {
		if err := m.store.AddEntry(entryName, retentionStoreEntry{
			Format:          format,
			InitialHoldTime: retainFor,
			Name:            timefmt.Format(time.Now(), format),
		}); !errors.Is(err, errDuplicateLabel) {
			addedLabels++
		}
	}

	if addedLabels == 0 {
		return ErrNoLabelsAdded
	}

	return nil
}

// CleanRetentions iterates all labels present and removes labels
// no longer covered by their retention duration
func (m Manager) CleanRetentions() {
	m.store.CleanupLabels(m.retention)
}

// GetClosestOlderBackup retrieves the backup closest to the given
// point in time but being created BEFORE that point in time
//
// Returns ErrNoBackupFound when there is no backup to return
func (m Manager) GetClosestOlderBackup(pointInTime time.Time) (string, error) {
	backup := m.store.FindRetainedBackupForPointInTime(pointInTime)
	if backup == "" {
		return "", ErrNoBackupFound
	}

	return backup, nil
}

// GetRetainedEntries lists all entries which match IsRetained
func (m Manager) GetRetainedEntries() []string {
	return m.store.ListRetainedEntries()
}

// GetUnretainedEntries lists all entries which do not match IsRetained
func (m Manager) GetUnretainedEntries() []string {
	return m.store.ListUnretainedEntries()
}

// IsKnown checks whether a backup is known to the Manager
func (m Manager) IsKnown(entryName string) bool {
	return m.store.IsEntryKnown(entryName)
}

// IsRetained checks whether there are still valid labels on the
// backup and therefore whether it should be retained or not.
// Before using IsRetained a CleanRetentions run should be executed
// in order to clean timed out labels.
func (m Manager) IsRetained(entryName string) bool {
	return m.store.IsEntryRetained(entryName)
}

// Remove removes an entry from the Manager causing IsKnown and
// IsRetained will return false afterwards
func (m Manager) Remove(entryName string) {
	m.store.Remove(entryName)
}

// Save stores the retention data to the given writer. You should
// take care this is an atomic write by writing into temp location
// and moving afterwards
func (m Manager) Save(dest io.Writer) error {
	return errors.Wrap(
		m.store.Save(dest),
		"saving store",
	)
}

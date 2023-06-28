package labelmanager

import (
	"io"
	"math"
	"sync"
	"time"

	"github.com/itchyny/timefmt-go"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type (
	retentionStore struct {
		Entries map[string][]retentionStoreEntry `yaml:"entries"`

		labels map[string]string
		lock   sync.RWMutex
	}

	retentionStoreEntry struct {
		Format          string        `yaml:"format"`
		InitialHoldTime time.Duration `yaml:"initialHoldTime"`
		Name            string        `yaml:"name"`
	}
)

var errDuplicateLabel = errors.New("label already exists")

func newRetentionStore() *retentionStore {
	return &retentionStore{
		Entries: make(map[string][]retentionStoreEntry),

		labels: make(map[string]string),
	}
}

// AddEntry checks whether the given label was already used
// and if not adds the label for the entry
//
// Returns errDuplicateLabel in case the label was already used
func (r *retentionStore) AddEntry(entry string, label retentionStoreEntry) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.rebuildLabels()

	if r.labels[label.Name] != "" {
		return errDuplicateLabel
	}

	if _, err := timefmt.Parse(label.Name, label.Format); err != nil {
		return errors.Wrap(err, "invalid entry name/format")
	}

	r.Entries[entry] = append(r.Entries[entry], label)
	return nil
}

// CleanupLabels removes timed out labels but does NOT delete the
// entry. This has to be done using the Remove function in
// order to give cleanup tasks the chance to see the is now
// no longer retained but was previously known.
func (r *retentionStore) CleanupLabels(retention RetentionConfig) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for entry, labels := range r.Entries {
		var retained []retentionStoreEntry
		for _, label := range labels {
			labelTime, err := timefmt.Parse(label.Name, label.Format)
			if err != nil {
				// We are checking entries on adding them, so this should not happen,
				// If it happens we treat the entry as invalid and drop it.
				continue
			}

			retainFor := label.InitialHoldTime
			if retention[label.Format] > 0 {
				// There is an entry in the current config which might
				// overwrite the initial hold time
				retainFor = retention[label.Format]
			}

			if labelTime.Add(retainFor).Before(time.Now()) {
				// That one expired, drop it.
				continue
			}

			retained = append(retained, label)
		}

		r.Entries[entry] = retained
	}

	r.rebuildLabels()
}

// FindRetainedBackupForPointInTime finds the closest OLDER backup than
// the given pointInTime, returns empty string if none matched the
// requirement
func (r *retentionStore) FindRetainedBackupForPointInTime(pointInTime time.Time) string {
	var (
		closest  string
		distance time.Duration = math.MaxInt64 // ~292.5 years
	)

	r.lock.RLock()
	defer r.lock.RUnlock()

	for entry, labels := range r.Entries {
		for _, label := range labels {
			labelTime, err := timefmt.Parse(label.Name, label.Format)
			if err != nil {
				// We are checking entries on adding them, so this should not happen,
				// If it happens we treat the entry as invalid and skip it.
				continue
			}

			if labelTime.After(pointInTime) {
				// We were asked for a backup older than pointInTime, this is not it.
				continue
			}

			dist := pointInTime.Sub(labelTime)
			if dist >= distance {
				// We already have a label closer to the pointInTime
				continue
			}

			// Yipp, that is better suited to be restored
			closest = entry
			distance = dist
		}
	}

	return closest
}

func (r *retentionStore) IsEntryKnown(entry string) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.Entries[entry] != nil
}

func (r *retentionStore) IsEntryRetained(entry string) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return len(r.Entries[entry]) > 0
}

func (r *retentionStore) ListRetainedEntries() []string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	var out []string
	for entry := range r.Entries {
		if r.IsEntryRetained(entry) {
			out = append(out, entry)
		}
	}

	return out
}

func (r *retentionStore) ListUnretainedEntries() []string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	var out []string
	for entry := range r.Entries {
		if !r.IsEntryRetained(entry) {
			out = append(out, entry)
		}
	}

	return out
}

// Load reads the retentionStore serialized to disk if one exists.
// If none exists no error will be reported but the store will not
// be modified.
func (r *retentionStore) Load(source io.Reader) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := yaml.NewDecoder(source).Decode(r); err != nil {
		return errors.Wrap(err, "reading store file")
	}

	r.rebuildLabels()

	return nil
}

// Remove deletes the entry from the database and rebuilds the labels
// list
func (r *retentionStore) Remove(entry string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.Entries, entry)
	r.rebuildLabels()
}

// Save serializes the store to the given writer. It is the users
// duty to make an atomic write out of it not to destroy data
func (r *retentionStore) Save(dest io.Writer) error {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return errors.Wrap(
		yaml.NewEncoder(dest).Encode(r),
		"encoding store",
	)
}

// rebuildLabels updates the label list for the entries currently
// known. It MUST NOT be used without previously acquiring a write-lock!
func (r *retentionStore) rebuildLabels() {
	for entry, labels := range r.Entries {
		for _, label := range labels {
			r.labels[label.Name] = entry
		}
	}
}

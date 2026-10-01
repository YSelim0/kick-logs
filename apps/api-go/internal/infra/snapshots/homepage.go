package snapshots

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

const maxSnapshotBytes = 1024 * 1024

type HomepageStore struct{ path string }

func NewHomepageStore(path string) *HomepageStore { return &HomepageStore{path: path} }

func (store *HomepageStore) Load() (domain.HomepageSnapshot, error) {
	file, err := os.Open(store.path)
	if err != nil {
		return domain.HomepageSnapshot{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSnapshotBytes+1))
	if err != nil {
		return domain.HomepageSnapshot{}, err
	}
	if len(data) > maxSnapshotBytes {
		return domain.HomepageSnapshot{}, errors.New("homepage snapshot exceeds size limit")
	}
	var snapshot domain.HomepageSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return domain.HomepageSnapshot{}, err
	}
	return snapshot, nil
}

func (store *HomepageStore) Save(snapshot domain.HomepageSnapshot) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	if len(data) > maxSnapshotBytes {
		return errors.New("homepage snapshot exceeds size limit")
	}
	directory := filepath.Dir(store.path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".homepage-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	// Same-directory rename never exposes a partially written snapshot to startup.
	return os.Rename(file.Name(), store.path)
}

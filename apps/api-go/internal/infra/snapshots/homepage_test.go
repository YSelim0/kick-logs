package snapshots

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

func TestHomepageStoreRoundTripAndReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "homepage.json")
	store := NewHomepageStore(path)
	if _, err := store.Load(); !os.IsNotExist(err) {
		t.Fatalf("missing cache: %v", err)
	}
	for _, value := range []int64{12, 25} {
		if err := store.Save(domain.HomepageSnapshot{Version: 1, Overview: domain.AnalyticsOverview{TotalMessages: value}}); err != nil {
			t.Fatal(err)
		}
		loaded, err := store.Load()
		if err != nil || loaded.Overview.TotalMessages != value {
			t.Fatalf("round trip = %+v, %v", loaded, err)
		}
	}
	files, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary files leaked: %v %v", files, err)
	}
}

func TestHomepageStoreRejectsCorruptAndOversizedFiles(t *testing.T) {
	for _, data := range [][]byte{[]byte("not json"), make([]byte, 1024*1024+1)} {
		path := filepath.Join(t.TempDir(), "homepage.json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewHomepageStore(path).Load(); err == nil {
			t.Fatal("accepted invalid cache")
		}
	}
}

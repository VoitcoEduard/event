package filestore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/postgres"
)

type stubMediaStore struct {
	existing map[string]bool
}

func (s *stubMediaStore) CreateMedia(context.Context, postgres.MediaRecord, *postgres.ScreensaverRecord, *postgres.TrackRecord) error {
	return nil
}
func (s *stubMediaStore) Media(context.Context, string) (postgres.MediaRecord, error) {
	return postgres.MediaRecord{}, nil
}
func (s *stubMediaStore) MediaStorageKeyExists(_ context.Context, key string) (bool, error) {
	return s.existing[key], nil
}
func (s *stubMediaStore) PendingMediaDeletion(context.Context) ([]postgres.MediaRecord, error) {
	return nil, nil
}
func (s *stubMediaStore) ForgetDeletedMedia(context.Context, string) error { return nil }
func (s *stubMediaStore) MarkMediaDeleted(context.Context, string) error   { return nil }
func (s *stubMediaStore) DeleteLibraryMedia(context.Context, string, string) (string, error) {
	return "", nil
}

func TestReconcileRemovesStaleProbeAndUploadTempFiles(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	oldTime := now.Add(-25 * time.Hour)

	staleProbe := filepath.Join(root, ".probe-old")
	staleUpload := filepath.Join(root, ".upload-old")
	freshProbe := filepath.Join(root, ".probe-fresh")
	for _, path := range []string{staleProbe, staleUpload, freshProbe} {
		if err := os.WriteFile(path, []byte("tmp"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{staleProbe, staleUpload} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
	}

	store := &Store{root: root, db: &stubMediaStore{existing: map[string]bool{}}}
	if err := store.Reconcile(context.Background(), now); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if _, err := os.Stat(staleProbe); !os.IsNotExist(err) {
		t.Fatalf("stale .probe file was not removed: %v", err)
	}
	if _, err := os.Stat(staleUpload); !os.IsNotExist(err) {
		t.Fatalf("stale .upload file was not removed: %v", err)
	}
	if _, err := os.Stat(freshProbe); err != nil {
		t.Fatalf("fresh .probe file should be kept: %v", err)
	}
}

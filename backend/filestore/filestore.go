// Package filestore persists uploaded media in a dedicated filesystem volume.
package filestore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/postgres"
)

type Store struct {
	root string
	db   *postgres.Store
}
type Upload struct {
	OriginalName string
	Filename     string
	MimeType     string
	EventID      string
	ModuleID     string
	Width        int
	Height       int
	Data         []byte
	// Kind is empty for event media, or "track"/"screensaver" for a library item.
	Kind string
	Name string
}
type Result struct {
	Media  postgres.MediaRecord
	Screen *postgres.ScreensaverRecord
	Track  *postgres.TrackRecord
}

func Open(root string, db *postgres.Store) (*Store, error) {
	if root == "" || db == nil {
		return nil, errors.New("media root and database are required")
	}
	path, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(path, 0700); err != nil {
		return nil, fmt.Errorf("create media root: %w", err)
	}
	s := &Store{root: path, db: db}
	if err = s.Ready(); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Store) Ready() error {
	f, err := os.CreateTemp(s.root, ".probe-*")
	if err != nil {
		return fmt.Errorf("media directory unavailable: %w", err)
	}
	name := f.Name()
	closeErr := f.Close()
	removeErr := os.Remove(name)
	return errors.Join(closeErr, removeErr)
}
func (s *Store) path(key string) (string, error) {
	parsed, err := uuid.Parse(key)
	if err != nil || parsed.String() != key {
		return "", errors.New("invalid storage key")
	}
	return filepath.Join(s.root, key), nil
}
func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (s *Store) Save(ctx context.Context, in Upload) (Result, error) {
	var out Result
	if in.Kind != "" && in.Kind != "track" && in.Kind != "screensaver" {
		return out, errors.New("invalid media kind")
	}
	id := uuid.NewString()
	path, err := s.path(id)
	if err != nil {
		return out, err
	}
	f, err := os.CreateTemp(s.root, ".upload-*")
	if err != nil {
		return out, err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(in.Data); err != nil {
		f.Close()
		return out, err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return out, err
	}
	if err = f.Close(); err != nil {
		return out, err
	}
	if err = os.Rename(tmp, path); err != nil {
		return out, err
	}
	if err = syncDir(s.root); err != nil {
		_ = os.Remove(path)
		return out, err
	}
	now := time.Now().UTC()
	out.Media = postgres.MediaRecord{ID: id, StorageKey: id, Filename: in.Filename, OriginalName: in.OriginalName, MimeType: in.MimeType, Size: int64(len(in.Data)), Width: in.Width, Height: in.Height, EventID: in.EventID, ModuleID: in.ModuleID, CreatedAt: now}
	if in.Kind == "track" {
		out.Track = &postgres.TrackRecord{ID: uuid.NewString(), Name: in.Name, MediaID: id, URL: "/media/" + id + "/file", CreatedAt: now}
	}
	if in.Kind == "screensaver" {
		out.Screen = &postgres.ScreensaverRecord{ID: uuid.NewString(), Name: in.Name, MediaID: id, ThumbnailURL: "/media/" + id + "/file", CreatedAt: now, UpdatedAt: now}
	}
	if err = s.db.CreateMedia(ctx, out.Media, out.Screen, out.Track); err != nil {
		// A failed commit can be ambiguous. Remove only when the database can
		// confirm no metadata exists; otherwise reconciliation handles orphans.
		checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		exists, checkErr := s.db.MediaStorageKeyExists(checkCtx, id)
		cancel()
		if checkErr == nil && !exists {
			_ = os.Remove(path)
		}
		return Result{}, err
	}
	return out, nil
}
func (s *Store) Read(ctx context.Context, id string) (postgres.MediaRecord, []byte, error) {
	m, err := s.db.Media(ctx, id)
	if err != nil {
		return m, nil, err
	}
	path, err := s.path(m.StorageKey)
	if err != nil {
		return m, nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return m, nil, err
	}
	return m, data, nil
}
func (s *Store) Delete(ctx context.Context, id string) error { return s.db.MarkMediaDeleted(ctx, id) }
func (s *Store) DeleteLibrary(ctx context.Context, kind, id string) (string, error) {
	return s.db.DeleteLibraryMedia(ctx, kind, id)
}

// Reconcile retries committed deletions and removes unreferenced files after a
// grace period, allowing interrupted writes and uncertain commits to settle.
func (s *Store) Reconcile(ctx context.Context, now time.Time) error {
	pending, err := s.db.PendingMediaDeletion(ctx)
	if err != nil {
		return err
	}
	for _, m := range pending {
		path, err := s.path(m.StorageKey)
		if err != nil {
			return err
		}
		err = os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err = s.db.ForgetDeletedMedia(ctx, m.ID); err != nil {
			return err
		}
	}
	files, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	for _, entry := range files {
		if err = ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || now.Sub(info.ModTime()) < 24*time.Hour {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".upload-") {
			if err = os.Remove(filepath.Join(s.root, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			continue
		}
		path, err := s.path(name)
		if err != nil {
			continue
		}
		exists, err := s.db.MediaStorageKeyExists(ctx, name)
		if err != nil {
			return err
		}
		if !exists {
			if err = os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}
func (s *Store) OpenFile(ctx context.Context, id string) (postgres.MediaRecord, *os.File, error) {
	m, err := s.db.Media(ctx, id)
	if err != nil {
		return m, nil, err
	}
	path, err := s.path(m.StorageKey)
	if err != nil {
		return m, nil, err
	}
	f, err := os.Open(path)
	return m, f, err
}

// DrainAndClose is useful when a client disconnects while receiving media.
func DrainAndClose(r io.ReadCloser) { _, _ = io.Copy(io.Discard, r); _ = r.Close() }

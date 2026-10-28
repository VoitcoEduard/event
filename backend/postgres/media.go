package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type MediaRecord struct {
	ID           string
	StorageKey   string
	Filename     string
	OriginalName string
	MimeType     string
	Size         int64
	Width        int
	Height       int
	EventID      string
	ModuleID     string
	CreatedAt    time.Time
}
type ScreensaverRecord struct {
	ID, Name, MediaID, ThumbnailURL string
	CreatedAt, UpdatedAt            time.Time
}
type TrackRecord struct {
	ID, Name, MediaID, URL string
	CreatedAt              time.Time
}

const mediaColumns = `id::text,storage_key,filename,original_name,mime_type,size,width,height,COALESCE(event_id::text,''),COALESCE(module_id::text,''),created_at`

func scanMedia(row scanner) (MediaRecord, error) {
	var m MediaRecord
	err := row.Scan(&m.ID, &m.StorageKey, &m.Filename, &m.OriginalName, &m.MimeType, &m.Size, &m.Width, &m.Height, &m.EventID, &m.ModuleID, &m.CreatedAt)
	return m, translate(err)
}

// CreateMedia inserts metadata and an optional library entry in one transaction.
func (s *Store) CreateMedia(ctx context.Context, m MediaRecord, screen *ScreensaverRecord, track *TrackRecord) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO media(id,storage_key,filename,original_name,mime_type,size,width,height,event_id,module_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid,NULLIF($10,'')::uuid,$11)`, m.ID, m.StorageKey, m.Filename, m.OriginalName, m.MimeType, m.Size, m.Width, m.Height, m.EventID, m.ModuleID, m.CreatedAt)
	if err != nil {
		return translate(err)
	}
	if screen != nil {
		if _, err = tx.Exec(ctx, `INSERT INTO screensavers(id,name,media_id,thumbnail_url,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6)`, screen.ID, screen.Name, m.ID, screen.ThumbnailURL, screen.CreatedAt, screen.UpdatedAt); err != nil {
			return translate(err)
		}
	}
	if track != nil {
		if _, err = tx.Exec(ctx, `INSERT INTO tracks(id,name,media_id,url,created_at) VALUES ($1,$2,$3,$4,$5)`, track.ID, track.Name, m.ID, track.URL, track.CreatedAt); err != nil {
			return translate(err)
		}
	}
	return translate(tx.Commit(ctx))
}
func (s *Store) Media(ctx context.Context, id string) (MediaRecord, error) {
	return scanMedia(s.Pool.QueryRow(ctx, `SELECT `+mediaColumns+` FROM media WHERE id=$1 AND deleted_at IS NULL`, id))
}
func (s *Store) MediaStorageKeyExists(ctx context.Context, key string) (bool, error) {
	var found bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM media WHERE storage_key=$1)`, key).Scan(&found)
	return found, translate(err)
}
func (s *Store) PendingMediaDeletion(ctx context.Context) ([]MediaRecord, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+mediaColumns+` FROM media WHERE deleted_at IS NOT NULL ORDER BY deleted_at,id LIMIT 100`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []MediaRecord{}
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) ForgetDeletedMedia(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM media WHERE id=$1 AND deleted_at IS NOT NULL`, id)
	return translate(err)
}
func (s *Store) MarkMediaDeleted(ctx context.Context, id string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE media SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err = affected(tag, err); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM screensavers WHERE media_id=$1`, id); err != nil {
		return translate(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM tracks WHERE media_id=$1`, id); err != nil {
		return translate(err)
	}
	return translate(tx.Commit(ctx))
}
func (s *Store) Screensavers(ctx context.Context) ([]ScreensaverRecord, error) {
	rows, err := s.Pool.Query(ctx, `SELECT s.id::text,s.name,s.media_id::text,s.thumbnail_url,s.created_at,s.updated_at FROM screensavers s JOIN media m ON m.id=s.media_id WHERE m.deleted_at IS NULL ORDER BY s.created_at,s.id`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []ScreensaverRecord{}
	for rows.Next() {
		var v ScreensaverRecord
		if err = rows.Scan(&v.ID, &v.Name, &v.MediaID, &v.ThumbnailURL, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) Tracks(ctx context.Context) ([]TrackRecord, error) {
	rows, err := s.Pool.Query(ctx, `SELECT t.id::text,t.name,t.media_id::text,t.url,t.created_at FROM tracks t JOIN media m ON m.id=t.media_id WHERE m.deleted_at IS NULL ORDER BY t.created_at,t.id`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []TrackRecord{}
	for rows.Next() {
		var v TrackRecord
		if err = rows.Scan(&v.ID, &v.Name, &v.MediaID, &v.URL, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) DeleteLibraryMedia(ctx context.Context, library, id string) (string, error) {
	var query string
	switch library {
	case "tracks":
		query = `DELETE FROM tracks WHERE id=$1 RETURNING media_id::text`
	case "screensavers":
		query = `DELETE FROM screensavers WHERE id=$1 RETURNING media_id::text`
	default:
		return "", ErrInvalidReference
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var mediaID string
	if err = tx.QueryRow(ctx, query, id).Scan(&mediaID); err != nil {
		return "", translate(err)
	}
	// Library files are owned by the entry. Remove any other library references
	// before marking the underlying file for asynchronous deletion.
	if _, err = tx.Exec(ctx, `DELETE FROM tracks WHERE media_id=$1`, mediaID); err != nil {
		return "", translate(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM screensavers WHERE media_id=$1`, mediaID); err != nil {
		return "", translate(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE media SET deleted_at=COALESCE(deleted_at,now()) WHERE id=$1`, mediaID); err != nil {
		return "", translate(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return "", translate(err)
	}
	return mediaID, nil
}

// Detach ownership before deleting the parent. Clearing both columns together
// avoids the event foreign key's SET NULL violating the module ownership check.
func detachOwnedMedia(ctx context.Context, tx pgx.Tx, eventID, moduleID string) ([]string, error) {
	rows, err := tx.Query(ctx, `UPDATE media SET
 event_id=CASE WHEN $2='' THEN NULL ELSE event_id END,module_id=NULL
 WHERE event_id=$1 AND ($2='' OR module_id=NULLIF($2,'')::uuid)
 RETURNING id::text`, eventID, moduleID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Templates and copied modules share media IDs. Retire detached files only
// after deleting the parent, so only surviving references keep them alive.
// Search nested JSON for IDs and media URLs to cover question images, picture
// options, slides and other configurable content without relying on its shape.
func retireUnreferencedMedia(ctx context.Context, tx pgx.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `UPDATE media m SET deleted_at=now()
 WHERE m.id::text=ANY($1::text[]) AND m.deleted_at IS NULL
 AND NOT EXISTS (SELECT 1 FROM templates t WHERE jsonb_path_exists(t.data,'$.** ? (@ == $id || @ == $url)',jsonb_build_object('id',m.id::text,'url','/media/'||m.id::text||'/file')))
 AND NOT EXISTS (SELECT 1 FROM modules v WHERE jsonb_path_exists(to_jsonb(v),'$.** ? (@ == $id || @ == $url)',jsonb_build_object('id',m.id::text,'url','/media/'||m.id::text||'/file')))
 AND NOT EXISTS (SELECT 1 FROM events e WHERE jsonb_path_exists(to_jsonb(e),'$.** ? (@ == $id || @ == $url)',jsonb_build_object('id',m.id::text,'url','/media/'||m.id::text||'/file')))
 AND NOT EXISTS (SELECT 1 FROM dynamic_cards c WHERE jsonb_path_exists(c.data,'$.** ? (@ == $id || @ == $url)',jsonb_build_object('id',m.id::text,'url','/media/'||m.id::text||'/file')))
 AND NOT EXISTS (SELECT 1 FROM screensavers s WHERE s.media_id=m.id)
 AND NOT EXISTS (SELECT 1 FROM tracks t WHERE t.media_id=m.id)`, ids)
	return translate(err)
}

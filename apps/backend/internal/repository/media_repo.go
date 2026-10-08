package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/1chooo/ad-service/internal/database"
	"github.com/1chooo/ad-service/internal/model"
)

type MediaRepository struct {
	db database.DB
}

func NewMediaRepository(db database.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) Create(ctx context.Context, media *model.Media) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO media (owner_id, storage_key, public_url, content_type, size_bytes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, media.OwnerID, media.StorageKey, media.URL, media.ContentType, media.SizeBytes).Scan(&media.ID, &media.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert media: %w", err)
	}
	return nil
}

func (r *MediaRepository) GetByIDAndOwner(ctx context.Context, id, ownerID int64) (*model.Media, error) {
	media, err := scanMedia(r.db.QueryRow(ctx, `
		SELECT id, owner_id, storage_key, public_url, content_type, size_bytes, created_at
		FROM media
		WHERE id = $1 AND owner_id = $2
	`, id, ownerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return media, nil
}

func (r *MediaRepository) DeleteByIDAndOwner(ctx context.Context, id, ownerID int64) (bool, error) {
	var deletedID int64
	err := r.db.QueryRow(ctx, `
		DELETE FROM media
		WHERE id = $1 AND owner_id = $2
		  AND NOT EXISTS (SELECT 1 FROM ads WHERE image_media_id = media.id)
		  AND NOT EXISTS (SELECT 1 FROM posts WHERE image_media_id = media.id)
		RETURNING id
	`, id, ownerID).Scan(&deletedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("delete media: %w", err)
	}
	return true, nil
}

func (r *MediaRepository) CanDeleteByIDAndOwner(ctx context.Context, id, ownerID int64) (bool, error) {
	var canDelete bool
	err := r.db.QueryRow(ctx, `
		SELECT NOT EXISTS (
			SELECT 1 FROM ads WHERE image_media_id = media.id
		) AND NOT EXISTS (SELECT 1 FROM posts WHERE image_media_id = media.id)
		FROM media
		WHERE id = $1 AND owner_id = $2
	`, id, ownerID).Scan(&canDelete)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check media usage: %w", err)
	}
	return canDelete, nil
}

// DeleteOwned holds the media row lock (or SQLite writer lock) until the object
// deletion finishes. Concurrent post/campaign references cannot pass their FK
// check between the usage check and removing the image bytes.
func (r *MediaRepository) DeleteOwned(ctx context.Context, id, ownerID int64, removeObject func(string) error) error {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var key string
	err = tx.QueryRow(ctx, `SELECT storage_key FROM media WHERE id = $1 AND owner_id = $2`+database.ForUpdate(tx), id, ownerID).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return model.NotFound("image not found")
	}
	if err != nil {
		return err
	}
	var referenced bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ads WHERE image_media_id = $1) OR EXISTS(SELECT 1 FROM posts WHERE image_media_id = $1)`, id).Scan(&referenced); err != nil {
		return err
	}
	if referenced {
		return model.Conflict("image is in use by a post or campaign")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM media WHERE id = $1 AND owner_id = $2`, id, ownerID); err != nil {
		return err
	}
	if err := removeObject(key); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func scanMedia(row rowScanner) (*model.Media, error) {
	var media model.Media
	if err := row.Scan(
		&media.ID,
		&media.OwnerID,
		&media.StorageKey,
		&media.URL,
		&media.ContentType,
		&media.SizeBytes,
		&media.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan media: %w", err)
	}
	return &media, nil
}

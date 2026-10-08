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
		)
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

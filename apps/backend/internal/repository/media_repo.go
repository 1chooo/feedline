package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MediaRepository struct {
	pool *pgxpool.Pool
}

func NewMediaRepository(pool *pgxpool.Pool) *MediaRepository {
	return &MediaRepository{pool: pool}
}

func (r *MediaRepository) Create(ctx context.Context, media *model.Media) error {
	err := r.pool.QueryRow(ctx, `
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
	media, err := scanMedia(r.pool.QueryRow(ctx, `
		SELECT id, owner_id, storage_key, public_url, content_type, size_bytes, created_at
		FROM media
		WHERE id = $1 AND owner_id = $2
	`, id, ownerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return media, nil
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("scan media: %w", err)
	}
	return &media, nil
}

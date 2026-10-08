package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/1chooo/ad-service/internal/storage"
)

type MediaStore interface {
	Create(ctx context.Context, media *model.Media) error
	GetByIDAndOwner(ctx context.Context, id, ownerID int64) (*model.Media, error)
	CanDeleteByIDAndOwner(ctx context.Context, id, ownerID int64) (bool, error)
	DeleteByIDAndOwner(ctx context.Context, id, ownerID int64) (bool, error)
	DeleteOwned(ctx context.Context, id, ownerID int64, removeObject func(string) error) error
}

type MediaService struct {
	store   MediaStore
	storage storage.ObjectStorage
	now     func() time.Time
}

func NewMediaService(store MediaStore, objectStorage storage.ObjectStorage) *MediaService {
	return &MediaService{store: store, storage: objectStorage, now: time.Now}
}

func (s *MediaService) UploadImage(ctx context.Context, ownerID int64, contentType string, body []byte) (*model.Media, error) {
	contentType, err := model.ValidateImage(contentType, int64(len(body)))
	if err != nil {
		return nil, err
	}

	key, err := imageKey(s.now().UTC(), contentType)
	if err != nil {
		return nil, err
	}
	publicURL, err := s.storage.Put(ctx, key, contentType, body)
	if err != nil {
		return nil, err
	}

	media := &model.Media{
		OwnerID:     ownerID,
		StorageKey:  key,
		URL:         publicURL,
		ContentType: contentType,
		SizeBytes:   int64(len(body)),
	}
	if err := s.store.Create(ctx, media); err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}
	return media, nil
}

func (s *MediaService) OwnedImage(ctx context.Context, ownerID, mediaID int64) (*model.Media, error) {
	if mediaID < 1 {
		return nil, model.NotFound("image not found")
	}
	media, err := s.store.GetByIDAndOwner(ctx, mediaID, ownerID)
	if err != nil {
		return nil, err
	}
	if media == nil {
		return nil, model.NotFound("image not found")
	}
	return media, nil
}

func (s *MediaService) DeleteImage(ctx context.Context, ownerID, mediaID int64) error {
	return s.store.DeleteOwned(ctx, mediaID, ownerID, func(key string) error { return s.storage.Delete(ctx, key) })
}

func imageKey(now time.Time, contentType string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate media key: %w", err)
	}
	return fmt.Sprintf("images/%04d/%02d/%s.%s", now.Year(), now.Month(), hex.EncodeToString(buf), model.ImageExtension(contentType)), nil
}

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/1chooo/ad-service/internal/model"
)

type memoryMediaStore struct {
	items []model.Media
}

func (m *memoryMediaStore) Create(_ context.Context, media *model.Media) error {
	media.ID = int64(len(m.items) + 1)
	media.CreatedAt = time.Now().UTC()
	m.items = append(m.items, *media)
	return nil
}

func (m *memoryMediaStore) GetByIDAndOwner(_ context.Context, id, ownerID int64) (*model.Media, error) {
	for _, media := range m.items {
		if media.ID == id && media.OwnerID == ownerID {
			copy := media
			return &copy, nil
		}
	}
	return nil, nil
}

func (m *memoryMediaStore) DeleteByIDAndOwner(_ context.Context, id, ownerID int64) (bool, error) {
	for index, media := range m.items {
		if media.ID == id && media.OwnerID == ownerID {
			m.items = append(m.items[:index], m.items[index+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (m *memoryMediaStore) CanDeleteByIDAndOwner(_ context.Context, id, ownerID int64) (bool, error) {
	for _, media := range m.items {
		if media.ID == id && media.OwnerID == ownerID {
			return true, nil
		}
	}
	return false, nil
}

func (m *memoryMediaStore) DeleteOwned(ctx context.Context, id, ownerID int64, removeObject func(string) error) error {
	media, err := m.GetByIDAndOwner(ctx, id, ownerID)
	if err != nil {
		return err
	}
	if media == nil {
		return model.NotFound("image not found")
	}
	if err := removeObject(media.StorageKey); err != nil {
		return err
	}
	_, err = m.DeleteByIDAndOwner(ctx, id, ownerID)
	return err
}

type memoryObjectStorage struct {
	key         string
	contentType string
	body        []byte
	deletedKey  string
}

func (s *memoryObjectStorage) Put(_ context.Context, key, contentType string, body []byte) (string, error) {
	s.key = key
	s.contentType = contentType
	s.body = append([]byte(nil), body...)
	return "https://media.example/" + key, nil
}

func (s *memoryObjectStorage) Delete(_ context.Context, key string) error {
	s.deletedKey = key
	return nil
}

func TestUploadImageStoresObjectAndMetadata(t *testing.T) {
	t.Parallel()

	store := &memoryMediaStore{}
	objects := &memoryObjectStorage{}
	svc := NewMediaService(store, objects)
	svc.now = func() time.Time { return time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC) }

	media, err := svc.UploadImage(context.Background(), 8, "image/png", []byte("fake png"))
	if err != nil {
		t.Fatalf("upload image: %v", err)
	}
	if media.OwnerID != 8 || media.URL == "" || media.SizeBytes != 8 {
		t.Fatalf("unexpected media metadata: %+v", media)
	}
	if !strings.HasPrefix(objects.key, "images/2026/10/") || !strings.HasSuffix(objects.key, ".png") {
		t.Fatalf("unexpected object key %q", objects.key)
	}
	if objects.contentType != "image/png" || string(objects.body) != "fake png" {
		t.Fatalf("unexpected object write: %+v", objects)
	}
}

func TestUploadImageRejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	svc := NewMediaService(&memoryMediaStore{}, &memoryObjectStorage{})
	_, err := svc.UploadImage(context.Background(), 1, "application/pdf", []byte("not an image"))
	if err == nil || err.Error() != "image must be a JPEG, PNG, WebP, or GIF" {
		t.Fatalf("expected unsupported image error, got %v", err)
	}
}

func TestDeleteImageRemovesObjectAndMetadata(t *testing.T) {
	t.Parallel()

	store := &memoryMediaStore{}
	objects := &memoryObjectStorage{}
	svc := NewMediaService(store, objects)
	media, err := svc.UploadImage(context.Background(), 3, "image/png", []byte("fake png"))
	if err != nil {
		t.Fatalf("UploadImage() error = %v", err)
	}
	if err := svc.DeleteImage(context.Background(), 3, media.ID); err != nil {
		t.Fatalf("DeleteImage() error = %v", err)
	}
	if objects.deletedKey != media.StorageKey {
		t.Fatalf("deleted key = %q, want %q", objects.deletedKey, media.StorageKey)
	}
	if len(store.items) != 0 {
		t.Fatalf("metadata records = %d, want 0", len(store.items))
	}
}

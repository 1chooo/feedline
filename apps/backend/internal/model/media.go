package model

import (
	"strings"
	"time"
)

const (
	MaxImageBytes = 5 << 20
)

var supportedImageTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
}

// Media contains the durable metadata for an object stored outside PostgreSQL.
// The image bytes themselves deliberately stay in object storage.
type Media struct {
	ID          int64     `json:"id"`
	OwnerID     int64     `json:"-"`
	StorageKey  string    `json:"-"`
	URL         string    `json:"url"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	CreatedAt   time.Time `json:"createdAt"`
}

func ValidateImage(contentType string, size int64) (string, error) {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if _, ok := supportedImageTypes[contentType]; !ok {
		return "", invalid("image must be a JPEG, PNG, WebP, or GIF")
	}
	if size < 1 {
		return "", invalid("image must not be empty")
	}
	if size > MaxImageBytes {
		return "", invalidf("image must be at most %d MB", MaxImageBytes>>20)
	}
	return contentType, nil
}

func ImageExtension(contentType string) string {
	return supportedImageTypes[contentType]
}

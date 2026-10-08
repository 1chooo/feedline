package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStoragePutAndDelete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store, err := NewLocal(dir, "http://localhost:8080/media")
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}
	key := "images/2026/10/creative.png"
	url, err := store.Put(context.Background(), key, "image/png", []byte("image"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if url != "http://localhost:8080/media/"+key {
		t.Fatalf("Put() URL = %q", url)
	}
	filename := filepath.Join(dir, "images", "2026", "10", "creative.png")
	if _, err := os.Stat(filename); err != nil {
		t.Fatalf("stored object stat error = %v", err)
	}
	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Fatalf("deleted object stat error = %v, want not exist", err)
	}
}

func TestLocalStorageDeleteRejectsUnsafeKey(t *testing.T) {
	t.Parallel()
	store, err := NewLocal(t.TempDir(), "http://localhost:8080/media")
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}
	if err := store.Delete(context.Background(), "../outside.png"); err == nil {
		t.Fatal("Delete() error = nil, want invalid key error")
	}
}

package knowledge

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystemObjectStorageSaveAndLoad(t *testing.T) {
	storage := NewFileSystemObjectStorage(t.TempDir())

	input := SaveObjectInput{
		Key:         "knowledge/org-1/kb-1/doc-1-vpn-guide.pdf",
		ContentType: "application/pdf",
		Content:     []byte("hello knowledge"),
	}

	if err := storage.Save(input); err != nil {
		t.Fatalf("expected save success, got error: %v", err)
	}

	object, err := storage.Load(input.Key)
	if err != nil {
		t.Fatalf("expected load success, got error: %v", err)
	}

	if object.Key != input.Key {
		t.Fatalf("expected key %q, got %q", input.Key, object.Key)
	}

	if object.ContentType != input.ContentType {
		t.Fatalf("expected content type %q, got %q", input.ContentType, object.ContentType)
	}

	if string(object.Content) != string(input.Content) {
		t.Fatalf("expected content %q, got %q", string(input.Content), string(object.Content))
	}
}

func TestFileSystemObjectStorageSaveCreatesDirectories(t *testing.T) {
	rootDir := filepath.Join(t.TempDir(), "nested-storage-root")
	storage := NewFileSystemObjectStorage(rootDir)
	key := "knowledge/org-1/kb-1/doc-1-vpn-guide.pdf"

	if err := storage.Save(SaveObjectInput{
		Key:         key,
		ContentType: "application/pdf",
		Content:     []byte("hello knowledge"),
	}); err != nil {
		t.Fatalf("expected save success, got error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(rootDir, key)); err != nil {
		t.Fatalf("expected saved object file to exist, got error: %v", err)
	}
}

func TestFileSystemObjectStorageLoadReturnsNotFound(t *testing.T) {
	storage := NewFileSystemObjectStorage(t.TempDir())

	_, err := storage.Load("knowledge/org-1/kb-1/missing.pdf")
	if !errors.Is(err, ErrKnowledgeObjectNotFound) {
		t.Fatalf("expected object not found error, got %v", err)
	}
}

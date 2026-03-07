package knowledge

import "testing"

func TestMemoryObjectStorageLoad(t *testing.T) {
	storage := NewMemoryObjectStorage()

	if err := storage.Save(SaveObjectInput{
		Key:         "documents/doc-1/vpn-guide.pdf",
		ContentType: "application/pdf",
		Content:     []byte("hello knowledge"),
	}); err != nil {
		t.Fatalf("save object: %v", err)
	}

	object, err := storage.Load("documents/doc-1/vpn-guide.pdf")
	if err != nil {
		t.Fatalf("load object: %v", err)
	}

	if object.Key != "documents/doc-1/vpn-guide.pdf" {
		t.Fatalf("expected key documents/doc-1/vpn-guide.pdf, got %q", object.Key)
	}
	if object.ContentType != "application/pdf" {
		t.Fatalf("expected content type application/pdf, got %q", object.ContentType)
	}
	if string(object.Content) != "hello knowledge" {
		t.Fatalf("expected content hello knowledge, got %q", string(object.Content))
	}
}

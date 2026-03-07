package knowledge

import "testing"

func TestDocumentDefaults(t *testing.T) {
	document := NewDocument(Document{
		KnowledgeBaseID: "kb-1",
		OrganizationID:  "org-1",
		Filename:        "vpn-guide.pdf",
		ContentType:     "application/pdf",
		SizeBytes:       1024,
		UploadedBy:      "user-agent-1",
	})

	if document.Status != DocumentStatusUploaded {
		t.Fatalf("expected default document status %q, got %q", DocumentStatusUploaded, document.Status)
	}

	if document.Filename != "vpn-guide.pdf" {
		t.Fatalf("expected filename vpn-guide.pdf, got %q", document.Filename)
	}

	knownStatuses := []DocumentStatus{
		DocumentStatusUploaded,
		DocumentStatusProcessing,
		DocumentStatusReady,
		DocumentStatusFailed,
	}

	for _, status := range knownStatuses {
		if status == "" {
			t.Fatalf("expected known document status to be non-empty")
		}
	}
}

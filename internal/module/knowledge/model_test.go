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

func TestProcessingTaskDefaults(t *testing.T) {
	task := NewDocumentProcessingTask(Document{
		ID:              "doc-1",
		KnowledgeBaseID: "kb-1",
		OrganizationID:  "org-1",
	}, 1, 3)

	if task.DocumentID != "doc-1" {
		t.Fatalf("expected document id doc-1, got %q", task.DocumentID)
	}
	if task.Status != ProcessingTaskStatusQueued {
		t.Fatalf("expected task status %q, got %q", ProcessingTaskStatusQueued, task.Status)
	}
	if task.Stage != ProcessingStageParse {
		t.Fatalf("expected task stage %q, got %q", ProcessingStageParse, task.Stage)
	}
	if task.Attempt != 1 {
		t.Fatalf("expected attempt 1, got %d", task.Attempt)
	}
	if task.MaxAttempts != 3 {
		t.Fatalf("expected max attempts 3, got %d", task.MaxAttempts)
	}
	if task.CreatedAt.IsZero() || task.UpdatedAt.IsZero() {
		t.Fatalf("expected task timestamps to be initialized")
	}
	if task.ID == "" {
		t.Fatalf("expected task id to be generated")
	}
}

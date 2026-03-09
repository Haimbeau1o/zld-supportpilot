package knowledge

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresKnowledgeBaseRepositorySaveFindAndListByOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresKnowledgeBaseRepository(db)
	createdAt := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	knowledgeBase := KnowledgeBase{
		OrganizationID: "org-1",
		Name:           "IT 支持知识库",
		Description:    "用于沉淀企业内部 IT 支持文档",
		CreatedBy:      "user-admin-1",
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO knowledge_bases (id, organization_id, name, description, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO UPDATE SET organization_id = EXCLUDED.organization_id, name = EXCLUDED.name, description = EXCLUDED.description, created_by = EXCLUDED.created_by, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at")).
		WithArgs(sqlmock.AnyArg(), knowledgeBase.OrganizationID, knowledgeBase.Name, knowledgeBase.Description, knowledgeBase.CreatedBy, knowledgeBase.CreatedAt, knowledgeBase.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(knowledgeBase)
	if saved.ID == "" {
		t.Fatalf("expected generated knowledge base id")
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, organization_id, name, description, created_by, created_at, updated_at FROM knowledge_bases WHERE id = $1")).
		WithArgs(saved.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "created_by", "created_at", "updated_at"}).AddRow(saved.ID, knowledgeBase.OrganizationID, knowledgeBase.Name, knowledgeBase.Description, knowledgeBase.CreatedBy, knowledgeBase.CreatedAt, knowledgeBase.UpdatedAt))

	stored, ok := repository.FindByID(saved.ID)
	if !ok {
		t.Fatalf("expected knowledge base to be found")
	}
	if stored.Name != knowledgeBase.Name {
		t.Fatalf("expected name %q, got %q", knowledgeBase.Name, stored.Name)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, organization_id, name, description, created_by, created_at, updated_at FROM knowledge_bases WHERE organization_id = $1 ORDER BY created_at ASC, id ASC")).
		WithArgs(knowledgeBase.OrganizationID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "created_by", "created_at", "updated_at"}).AddRow(saved.ID, knowledgeBase.OrganizationID, knowledgeBase.Name, knowledgeBase.Description, knowledgeBase.CreatedBy, knowledgeBase.CreatedAt, knowledgeBase.UpdatedAt))

	items := repository.ListByOrganization(knowledgeBase.OrganizationID)
	if len(items) != 1 {
		t.Fatalf("expected 1 knowledge base, got %d", len(items))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPostgresDocumentRepositorySaveFindAndListByKnowledgeBase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresDocumentRepository(db)
	createdAt := time.Date(2026, 3, 9, 13, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	document := Document{
		KnowledgeBaseID:     "kb-1",
		OrganizationID:      "org-1",
		Filename:            "vpn-guide.pdf",
		ContentType:         "application/pdf",
		SizeBytes:           128,
		StorageKey:          "knowledge/org-1/kb-1/doc-1-vpn-guide.pdf",
		Status:              DocumentStatusUploaded,
		UploadedBy:          "user-admin-1",
		LastTaskID:          "",
		ProcessingAttempts:  0,
		ChunkCount:          0,
		LastError:           "",
		ProcessingQueuedAt:  time.Time{},
		ProcessingStartedAt: time.Time{},
		ProcessedAt:         time.Time{},
		FailedAt:            time.Time{},
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO knowledge_documents (id, knowledge_base_id, organization_id, filename, content_type, size_bytes, storage_key, status, uploaded_by, last_task_id, processing_attempts, chunk_count, last_error, processing_queued_at, processing_started_at, processed_at, failed_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19) ON CONFLICT (id) DO UPDATE SET knowledge_base_id = EXCLUDED.knowledge_base_id, organization_id = EXCLUDED.organization_id, filename = EXCLUDED.filename, content_type = EXCLUDED.content_type, size_bytes = EXCLUDED.size_bytes, storage_key = EXCLUDED.storage_key, status = EXCLUDED.status, uploaded_by = EXCLUDED.uploaded_by, last_task_id = EXCLUDED.last_task_id, processing_attempts = EXCLUDED.processing_attempts, chunk_count = EXCLUDED.chunk_count, last_error = EXCLUDED.last_error, processing_queued_at = EXCLUDED.processing_queued_at, processing_started_at = EXCLUDED.processing_started_at, processed_at = EXCLUDED.processed_at, failed_at = EXCLUDED.failed_at, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at")).
		WithArgs(sqlmock.AnyArg(), document.KnowledgeBaseID, document.OrganizationID, document.Filename, document.ContentType, document.SizeBytes, document.StorageKey, string(document.Status), document.UploadedBy, document.LastTaskID, document.ProcessingAttempts, document.ChunkCount, document.LastError, nil, nil, nil, nil, document.CreatedAt, document.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(document)
	if saved.ID == "" {
		t.Fatalf("expected generated document id")
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, knowledge_base_id, organization_id, filename, content_type, size_bytes, storage_key, status, uploaded_by, last_task_id, processing_attempts, chunk_count, last_error, processing_queued_at, processing_started_at, processed_at, failed_at, created_at, updated_at FROM knowledge_documents WHERE id = $1")).
		WithArgs(saved.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "knowledge_base_id", "organization_id", "filename", "content_type", "size_bytes", "storage_key", "status", "uploaded_by", "last_task_id", "processing_attempts", "chunk_count", "last_error", "processing_queued_at", "processing_started_at", "processed_at", "failed_at", "created_at", "updated_at"}).AddRow(saved.ID, document.KnowledgeBaseID, document.OrganizationID, document.Filename, document.ContentType, document.SizeBytes, document.StorageKey, string(document.Status), document.UploadedBy, document.LastTaskID, document.ProcessingAttempts, document.ChunkCount, document.LastError, nil, nil, nil, nil, document.CreatedAt, document.UpdatedAt))

	stored, ok := repository.FindByID(saved.ID)
	if !ok {
		t.Fatalf("expected document to be found")
	}
	if stored.StorageKey != document.StorageKey {
		t.Fatalf("expected storage key %q, got %q", document.StorageKey, stored.StorageKey)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, knowledge_base_id, organization_id, filename, content_type, size_bytes, storage_key, status, uploaded_by, last_task_id, processing_attempts, chunk_count, last_error, processing_queued_at, processing_started_at, processed_at, failed_at, created_at, updated_at FROM knowledge_documents WHERE knowledge_base_id = $1 ORDER BY created_at ASC, id ASC")).
		WithArgs(document.KnowledgeBaseID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "knowledge_base_id", "organization_id", "filename", "content_type", "size_bytes", "storage_key", "status", "uploaded_by", "last_task_id", "processing_attempts", "chunk_count", "last_error", "processing_queued_at", "processing_started_at", "processed_at", "failed_at", "created_at", "updated_at"}).AddRow(saved.ID, document.KnowledgeBaseID, document.OrganizationID, document.Filename, document.ContentType, document.SizeBytes, document.StorageKey, string(document.Status), document.UploadedBy, document.LastTaskID, document.ProcessingAttempts, document.ChunkCount, document.LastError, nil, nil, nil, nil, document.CreatedAt, document.UpdatedAt))

	items := repository.ListByKnowledgeBase(document.KnowledgeBaseID)
	if len(items) != 1 {
		t.Fatalf("expected 1 document, got %d", len(items))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPostgresDocumentProcessingTaskRepositorySaveFindLatestAndListByDocument(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresDocumentProcessingTaskRepository(db)
	createdAt := time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	task := DocumentProcessingTask{
		ID:              "doc-1-process-1",
		DocumentID:      "doc-1",
		KnowledgeBaseID: "kb-1",
		OrganizationID:  "org-1",
		Status:          ProcessingTaskStatusQueued,
		Stage:           ProcessingStageParse,
		Attempt:         1,
		MaxAttempts:     3,
		ErrorMessage:    "",
		CreatedAt:       createdAt,
		StartedAt:       time.Time{},
		FinishedAt:      time.Time{},
		UpdatedAt:       updatedAt,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO knowledge_processing_tasks (id, document_id, knowledge_base_id, organization_id, status, stage, attempt, max_attempts, error_message, created_at, started_at, finished_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT (id) DO UPDATE SET document_id = EXCLUDED.document_id, knowledge_base_id = EXCLUDED.knowledge_base_id, organization_id = EXCLUDED.organization_id, status = EXCLUDED.status, stage = EXCLUDED.stage, attempt = EXCLUDED.attempt, max_attempts = EXCLUDED.max_attempts, error_message = EXCLUDED.error_message, created_at = EXCLUDED.created_at, started_at = EXCLUDED.started_at, finished_at = EXCLUDED.finished_at, updated_at = EXCLUDED.updated_at")).
		WithArgs(task.ID, task.DocumentID, task.KnowledgeBaseID, task.OrganizationID, string(task.Status), string(task.Stage), task.Attempt, task.MaxAttempts, task.ErrorMessage, task.CreatedAt, nil, nil, task.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(task)
	if saved.ID != task.ID {
		t.Fatalf("expected task id %q, got %q", task.ID, saved.ID)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, document_id, knowledge_base_id, organization_id, status, stage, attempt, max_attempts, error_message, created_at, started_at, finished_at, updated_at FROM knowledge_processing_tasks WHERE id = $1")).
		WithArgs(task.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "knowledge_base_id", "organization_id", "status", "stage", "attempt", "max_attempts", "error_message", "created_at", "started_at", "finished_at", "updated_at"}).AddRow(task.ID, task.DocumentID, task.KnowledgeBaseID, task.OrganizationID, string(task.Status), string(task.Stage), task.Attempt, task.MaxAttempts, task.ErrorMessage, task.CreatedAt, nil, nil, task.UpdatedAt))

	stored, ok := repository.FindByID(task.ID)
	if !ok {
		t.Fatalf("expected task to be found")
	}
	if stored.Stage != task.Stage {
		t.Fatalf("expected stage %q, got %q", task.Stage, stored.Stage)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, document_id, knowledge_base_id, organization_id, status, stage, attempt, max_attempts, error_message, created_at, started_at, finished_at, updated_at FROM knowledge_processing_tasks WHERE document_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1")).
		WithArgs(task.DocumentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "knowledge_base_id", "organization_id", "status", "stage", "attempt", "max_attempts", "error_message", "created_at", "started_at", "finished_at", "updated_at"}).AddRow(task.ID, task.DocumentID, task.KnowledgeBaseID, task.OrganizationID, string(task.Status), string(task.Stage), task.Attempt, task.MaxAttempts, task.ErrorMessage, task.CreatedAt, nil, nil, task.UpdatedAt))

	latest, ok := repository.FindLatestByDocument(task.DocumentID)
	if !ok {
		t.Fatalf("expected latest task to be found")
	}
	if latest.ID != task.ID {
		t.Fatalf("expected latest task id %q, got %q", task.ID, latest.ID)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, document_id, knowledge_base_id, organization_id, status, stage, attempt, max_attempts, error_message, created_at, started_at, finished_at, updated_at FROM knowledge_processing_tasks WHERE document_id = $1 ORDER BY created_at ASC, id ASC")).
		WithArgs(task.DocumentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "knowledge_base_id", "organization_id", "status", "stage", "attempt", "max_attempts", "error_message", "created_at", "started_at", "finished_at", "updated_at"}).AddRow(task.ID, task.DocumentID, task.KnowledgeBaseID, task.OrganizationID, string(task.Status), string(task.Stage), task.Attempt, task.MaxAttempts, task.ErrorMessage, task.CreatedAt, nil, nil, task.UpdatedAt))

	items := repository.ListByDocument(task.DocumentID)
	if len(items) != 1 {
		t.Fatalf("expected 1 task, got %d", len(items))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPostgresDocumentChunkRepositoryReplaceAndList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresDocumentChunkRepository(db)
	createdAt := time.Date(2026, 3, 9, 15, 0, 0, 0, time.UTC)
	chunks := []DocumentChunk{
		{
			ID:              "doc-1-chunk-1",
			DocumentID:      "doc-1",
			KnowledgeBaseID: "kb-1",
			OrganizationID:  "org-1",
			Sequence:        1,
			Content:         "chunk-1",
			CreatedAt:       createdAt,
		},
		{
			ID:              "doc-1-chunk-2",
			DocumentID:      "doc-1",
			KnowledgeBaseID: "kb-1",
			OrganizationID:  "org-1",
			Sequence:        2,
			Content:         "chunk-2",
			CreatedAt:       createdAt.Add(time.Minute),
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM knowledge_document_chunks WHERE document_id = $1")).
		WithArgs("doc-1").
		WillReturnResult(sqlmock.NewResult(1, 2))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO knowledge_document_chunks (id, document_id, knowledge_base_id, organization_id, sequence, content, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)")).
		WithArgs(chunks[0].ID, chunks[0].DocumentID, chunks[0].KnowledgeBaseID, chunks[0].OrganizationID, chunks[0].Sequence, chunks[0].Content, chunks[0].CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO knowledge_document_chunks (id, document_id, knowledge_base_id, organization_id, sequence, content, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)")).
		WithArgs(chunks[1].ID, chunks[1].DocumentID, chunks[1].KnowledgeBaseID, chunks[1].OrganizationID, chunks[1].Sequence, chunks[1].Content, chunks[1].CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repository.ReplaceByDocument("doc-1", chunks)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, document_id, knowledge_base_id, organization_id, sequence, content, created_at FROM knowledge_document_chunks WHERE document_id = $1 ORDER BY sequence ASC, id ASC")).
		WithArgs("doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "knowledge_base_id", "organization_id", "sequence", "content", "created_at"}).
			AddRow(chunks[0].ID, chunks[0].DocumentID, chunks[0].KnowledgeBaseID, chunks[0].OrganizationID, chunks[0].Sequence, chunks[0].Content, chunks[0].CreatedAt).
			AddRow(chunks[1].ID, chunks[1].DocumentID, chunks[1].KnowledgeBaseID, chunks[1].OrganizationID, chunks[1].Sequence, chunks[1].Content, chunks[1].CreatedAt))

	documentChunks := repository.ListByDocument("doc-1")
	if len(documentChunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(documentChunks))
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, document_id, knowledge_base_id, organization_id, sequence, content, created_at FROM knowledge_document_chunks WHERE knowledge_base_id = $1 ORDER BY created_at ASC, id ASC")).
		WithArgs("kb-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "knowledge_base_id", "organization_id", "sequence", "content", "created_at"}).
			AddRow(chunks[0].ID, chunks[0].DocumentID, chunks[0].KnowledgeBaseID, chunks[0].OrganizationID, chunks[0].Sequence, chunks[0].Content, chunks[0].CreatedAt).
			AddRow(chunks[1].ID, chunks[1].DocumentID, chunks[1].KnowledgeBaseID, chunks[1].OrganizationID, chunks[1].Sequence, chunks[1].Content, chunks[1].CreatedAt))

	knowledgeBaseChunks := repository.ListByKnowledgeBase("kb-1")
	if len(knowledgeBaseChunks) != 2 {
		t.Fatalf("expected 2 knowledge base chunks, got %d", len(knowledgeBaseChunks))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

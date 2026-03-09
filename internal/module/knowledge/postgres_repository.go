package knowledge

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/persistence"
)

// PostgresKnowledgeBaseRepository 使用 PostgreSQL 持久化知识库元数据。
type PostgresKnowledgeBaseRepository struct {
	db *sql.DB
}

func NewPostgresKnowledgeBaseRepository(db *sql.DB) *PostgresKnowledgeBaseRepository {
	return &PostgresKnowledgeBaseRepository{db: db}
}

func (repository *PostgresKnowledgeBaseRepository) Save(knowledgeBase KnowledgeBase) KnowledgeBase {
	if knowledgeBase.ID == "" {
		knowledgeBase.ID = persistence.NewID("kb")
	}

	if _, err := repository.db.Exec(
		`INSERT INTO knowledge_bases (id, organization_id, name, description, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO UPDATE SET organization_id = EXCLUDED.organization_id, name = EXCLUDED.name, description = EXCLUDED.description, created_by = EXCLUDED.created_by, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at`,
		knowledgeBase.ID,
		knowledgeBase.OrganizationID,
		knowledgeBase.Name,
		knowledgeBase.Description,
		knowledgeBase.CreatedBy,
		knowledgeBase.CreatedAt,
		knowledgeBase.UpdatedAt,
	); err != nil {
		panic(fmt.Sprintf("save knowledge base to postgres: %v", err))
	}

	return knowledgeBase
}

func (repository *PostgresKnowledgeBaseRepository) FindByID(knowledgeBaseID string) (KnowledgeBase, bool) {
	row := repository.db.QueryRow(`SELECT id, organization_id, name, description, created_by, created_at, updated_at FROM knowledge_bases WHERE id = $1`, knowledgeBaseID)
	return scanKnowledgeBase(row)
}

func (repository *PostgresKnowledgeBaseRepository) ListByOrganization(organizationID string) []KnowledgeBase {
	rows, err := repository.db.Query(`SELECT id, organization_id, name, description, created_by, created_at, updated_at FROM knowledge_bases WHERE organization_id = $1 ORDER BY created_at ASC, id ASC`, organizationID)
	if err != nil {
		panic(fmt.Sprintf("list knowledge bases from postgres: %v", err))
	}
	defer rows.Close()

	items := make([]KnowledgeBase, 0)
	for rows.Next() {
		item, ok := scanKnowledgeBase(rows)
		if ok {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate knowledge bases from postgres: %v", err))
	}

	return items
}

// PostgresDocumentRepository 使用 PostgreSQL 持久化文档元数据。
type PostgresDocumentRepository struct {
	db *sql.DB
}

func NewPostgresDocumentRepository(db *sql.DB) *PostgresDocumentRepository {
	return &PostgresDocumentRepository{db: db}
}

func (repository *PostgresDocumentRepository) Save(document Document) Document {
	if document.ID == "" {
		document.ID = persistence.NewID("doc")
	}

	if _, err := repository.db.Exec(
		`INSERT INTO knowledge_documents (id, knowledge_base_id, organization_id, filename, content_type, size_bytes, storage_key, status, uploaded_by, last_task_id, processing_attempts, chunk_count, last_error, processing_queued_at, processing_started_at, processed_at, failed_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19) ON CONFLICT (id) DO UPDATE SET knowledge_base_id = EXCLUDED.knowledge_base_id, organization_id = EXCLUDED.organization_id, filename = EXCLUDED.filename, content_type = EXCLUDED.content_type, size_bytes = EXCLUDED.size_bytes, storage_key = EXCLUDED.storage_key, status = EXCLUDED.status, uploaded_by = EXCLUDED.uploaded_by, last_task_id = EXCLUDED.last_task_id, processing_attempts = EXCLUDED.processing_attempts, chunk_count = EXCLUDED.chunk_count, last_error = EXCLUDED.last_error, processing_queued_at = EXCLUDED.processing_queued_at, processing_started_at = EXCLUDED.processing_started_at, processed_at = EXCLUDED.processed_at, failed_at = EXCLUDED.failed_at, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at`,
		document.ID,
		document.KnowledgeBaseID,
		document.OrganizationID,
		document.Filename,
		document.ContentType,
		document.SizeBytes,
		document.StorageKey,
		string(document.Status),
		document.UploadedBy,
		document.LastTaskID,
		document.ProcessingAttempts,
		document.ChunkCount,
		document.LastError,
		persistence.NullableTime(document.ProcessingQueuedAt),
		persistence.NullableTime(document.ProcessingStartedAt),
		persistence.NullableTime(document.ProcessedAt),
		persistence.NullableTime(document.FailedAt),
		document.CreatedAt,
		document.UpdatedAt,
	); err != nil {
		panic(fmt.Sprintf("save document to postgres: %v", err))
	}

	return document
}

func (repository *PostgresDocumentRepository) FindByID(documentID string) (Document, bool) {
	row := repository.db.QueryRow(`SELECT id, knowledge_base_id, organization_id, filename, content_type, size_bytes, storage_key, status, uploaded_by, last_task_id, processing_attempts, chunk_count, last_error, processing_queued_at, processing_started_at, processed_at, failed_at, created_at, updated_at FROM knowledge_documents WHERE id = $1`, documentID)
	return scanDocument(row)
}

func (repository *PostgresDocumentRepository) ListByKnowledgeBase(knowledgeBaseID string) []Document {
	rows, err := repository.db.Query(`SELECT id, knowledge_base_id, organization_id, filename, content_type, size_bytes, storage_key, status, uploaded_by, last_task_id, processing_attempts, chunk_count, last_error, processing_queued_at, processing_started_at, processed_at, failed_at, created_at, updated_at FROM knowledge_documents WHERE knowledge_base_id = $1 ORDER BY created_at ASC, id ASC`, knowledgeBaseID)
	if err != nil {
		panic(fmt.Sprintf("list documents from postgres: %v", err))
	}
	defer rows.Close()

	items := make([]Document, 0)
	for rows.Next() {
		item, ok := scanDocument(rows)
		if ok {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate documents from postgres: %v", err))
	}

	return items
}

// PostgresDocumentProcessingTaskRepository 使用 PostgreSQL 持久化处理任务轨迹。
type PostgresDocumentProcessingTaskRepository struct {
	db *sql.DB
}

func NewPostgresDocumentProcessingTaskRepository(db *sql.DB) *PostgresDocumentProcessingTaskRepository {
	return &PostgresDocumentProcessingTaskRepository{db: db}
}

func (repository *PostgresDocumentProcessingTaskRepository) Save(task DocumentProcessingTask) DocumentProcessingTask {
	if _, err := repository.db.Exec(
		`INSERT INTO knowledge_processing_tasks (id, document_id, knowledge_base_id, organization_id, status, stage, attempt, max_attempts, error_message, created_at, started_at, finished_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT (id) DO UPDATE SET document_id = EXCLUDED.document_id, knowledge_base_id = EXCLUDED.knowledge_base_id, organization_id = EXCLUDED.organization_id, status = EXCLUDED.status, stage = EXCLUDED.stage, attempt = EXCLUDED.attempt, max_attempts = EXCLUDED.max_attempts, error_message = EXCLUDED.error_message, created_at = EXCLUDED.created_at, started_at = EXCLUDED.started_at, finished_at = EXCLUDED.finished_at, updated_at = EXCLUDED.updated_at`,
		task.ID,
		task.DocumentID,
		task.KnowledgeBaseID,
		task.OrganizationID,
		string(task.Status),
		string(task.Stage),
		task.Attempt,
		task.MaxAttempts,
		task.ErrorMessage,
		task.CreatedAt,
		persistence.NullableTime(task.StartedAt),
		persistence.NullableTime(task.FinishedAt),
		task.UpdatedAt,
	); err != nil {
		panic(fmt.Sprintf("save processing task to postgres: %v", err))
	}

	return task
}

func (repository *PostgresDocumentProcessingTaskRepository) FindByID(taskID string) (DocumentProcessingTask, bool) {
	row := repository.db.QueryRow(`SELECT id, document_id, knowledge_base_id, organization_id, status, stage, attempt, max_attempts, error_message, created_at, started_at, finished_at, updated_at FROM knowledge_processing_tasks WHERE id = $1`, taskID)
	return scanProcessingTask(row)
}

func (repository *PostgresDocumentProcessingTaskRepository) FindLatestByDocument(documentID string) (DocumentProcessingTask, bool) {
	row := repository.db.QueryRow(`SELECT id, document_id, knowledge_base_id, organization_id, status, stage, attempt, max_attempts, error_message, created_at, started_at, finished_at, updated_at FROM knowledge_processing_tasks WHERE document_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, documentID)
	return scanProcessingTask(row)
}

func (repository *PostgresDocumentProcessingTaskRepository) ListByDocument(documentID string) []DocumentProcessingTask {
	rows, err := repository.db.Query(`SELECT id, document_id, knowledge_base_id, organization_id, status, stage, attempt, max_attempts, error_message, created_at, started_at, finished_at, updated_at FROM knowledge_processing_tasks WHERE document_id = $1 ORDER BY created_at ASC, id ASC`, documentID)
	if err != nil {
		panic(fmt.Sprintf("list processing tasks from postgres: %v", err))
	}
	defer rows.Close()

	items := make([]DocumentProcessingTask, 0)
	for rows.Next() {
		item, ok := scanProcessingTask(rows)
		if ok {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate processing tasks from postgres: %v", err))
	}

	return items
}

// PostgresDocumentChunkRepository 使用 PostgreSQL 持久化切块结果。
type PostgresDocumentChunkRepository struct {
	db *sql.DB
}

func NewPostgresDocumentChunkRepository(db *sql.DB) *PostgresDocumentChunkRepository {
	return &PostgresDocumentChunkRepository{db: db}
}

func (repository *PostgresDocumentChunkRepository) ReplaceByDocument(documentID string, chunks []DocumentChunk) {
	tx, err := repository.db.Begin()
	if err != nil {
		panic(fmt.Sprintf("begin replace chunks transaction: %v", err))
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM knowledge_document_chunks WHERE document_id = $1`, documentID); err != nil {
		panic(fmt.Sprintf("delete chunks by document from postgres: %v", err))
	}

	for _, chunk := range chunks {
		if _, err := tx.Exec(
			`INSERT INTO knowledge_document_chunks (id, document_id, knowledge_base_id, organization_id, sequence, content, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			chunk.ID,
			chunk.DocumentID,
			chunk.KnowledgeBaseID,
			chunk.OrganizationID,
			chunk.Sequence,
			chunk.Content,
			chunk.CreatedAt,
		); err != nil {
			panic(fmt.Sprintf("insert chunk to postgres: %v", err))
		}
	}

	if err := tx.Commit(); err != nil {
		panic(fmt.Sprintf("commit replace chunks transaction: %v", err))
	}
}

func (repository *PostgresDocumentChunkRepository) ListByDocument(documentID string) []DocumentChunk {
	rows, err := repository.db.Query(`SELECT id, document_id, knowledge_base_id, organization_id, sequence, content, created_at FROM knowledge_document_chunks WHERE document_id = $1 ORDER BY sequence ASC, id ASC`, documentID)
	if err != nil {
		panic(fmt.Sprintf("list chunks by document from postgres: %v", err))
	}
	defer rows.Close()

	items := make([]DocumentChunk, 0)
	for rows.Next() {
		chunk := DocumentChunk{}
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.KnowledgeBaseID, &chunk.OrganizationID, &chunk.Sequence, &chunk.Content, &chunk.CreatedAt); err != nil {
			panic(fmt.Sprintf("scan chunk from postgres: %v", err))
		}
		items = append(items, chunk)
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate chunks by document from postgres: %v", err))
	}

	return items
}

func (repository *PostgresDocumentChunkRepository) ListByKnowledgeBase(knowledgeBaseID string) []DocumentChunk {
	rows, err := repository.db.Query(`SELECT id, document_id, knowledge_base_id, organization_id, sequence, content, created_at FROM knowledge_document_chunks WHERE knowledge_base_id = $1 ORDER BY created_at ASC, id ASC`, knowledgeBaseID)
	if err != nil {
		panic(fmt.Sprintf("list chunks by knowledge base from postgres: %v", err))
	}
	defer rows.Close()

	items := make([]DocumentChunk, 0)
	for rows.Next() {
		chunk := DocumentChunk{}
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.KnowledgeBaseID, &chunk.OrganizationID, &chunk.Sequence, &chunk.Content, &chunk.CreatedAt); err != nil {
			panic(fmt.Sprintf("scan chunk from postgres: %v", err))
		}
		items = append(items, chunk)
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate chunks by knowledge base from postgres: %v", err))
	}

	return items
}

type knowledgeRowScanner interface {
	Scan(dest ...any) error
}

func scanKnowledgeBase(scanner knowledgeRowScanner) (KnowledgeBase, bool) {
	knowledgeBase := KnowledgeBase{}
	if err := scanner.Scan(&knowledgeBase.ID, &knowledgeBase.OrganizationID, &knowledgeBase.Name, &knowledgeBase.Description, &knowledgeBase.CreatedBy, &knowledgeBase.CreatedAt, &knowledgeBase.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return KnowledgeBase{}, false
		}
		panic(fmt.Sprintf("scan knowledge base from postgres: %v", err))
	}
	return knowledgeBase, true
}

func scanDocument(scanner knowledgeRowScanner) (Document, bool) {
	document := Document{}
	var status string
	var processingQueuedAt sql.NullTime
	var processingStartedAt sql.NullTime
	var processedAt sql.NullTime
	var failedAt sql.NullTime
	if err := scanner.Scan(
		&document.ID,
		&document.KnowledgeBaseID,
		&document.OrganizationID,
		&document.Filename,
		&document.ContentType,
		&document.SizeBytes,
		&document.StorageKey,
		&status,
		&document.UploadedBy,
		&document.LastTaskID,
		&document.ProcessingAttempts,
		&document.ChunkCount,
		&document.LastError,
		&processingQueuedAt,
		&processingStartedAt,
		&processedAt,
		&failedAt,
		&document.CreatedAt,
		&document.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Document{}, false
		}
		panic(fmt.Sprintf("scan document from postgres: %v", err))
	}

	document.Status = DocumentStatus(status)
	document.ProcessingQueuedAt = persistence.FromNullableTime(processingQueuedAt)
	document.ProcessingStartedAt = persistence.FromNullableTime(processingStartedAt)
	document.ProcessedAt = persistence.FromNullableTime(processedAt)
	document.FailedAt = persistence.FromNullableTime(failedAt)
	return document, true
}

func scanProcessingTask(scanner knowledgeRowScanner) (DocumentProcessingTask, bool) {
	task := DocumentProcessingTask{}
	var status string
	var stage string
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	if err := scanner.Scan(
		&task.ID,
		&task.DocumentID,
		&task.KnowledgeBaseID,
		&task.OrganizationID,
		&status,
		&stage,
		&task.Attempt,
		&task.MaxAttempts,
		&task.ErrorMessage,
		&task.CreatedAt,
		&startedAt,
		&finishedAt,
		&task.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DocumentProcessingTask{}, false
		}
		panic(fmt.Sprintf("scan processing task from postgres: %v", err))
	}
	task.Status = ProcessingTaskStatus(status)
	task.Stage = ProcessingStage(stage)
	task.StartedAt = persistence.FromNullableTime(startedAt)
	task.FinishedAt = persistence.FromNullableTime(finishedAt)
	return task, true
}

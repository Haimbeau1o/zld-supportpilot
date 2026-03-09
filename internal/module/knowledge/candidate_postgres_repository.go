package knowledge

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/persistence"
)

// PostgresKnowledgeCandidateRepository 使用 PostgreSQL 持久化知识候选元数据。
type PostgresKnowledgeCandidateRepository struct {
	db *sql.DB
}

func NewPostgresKnowledgeCandidateRepository(db *sql.DB) *PostgresKnowledgeCandidateRepository {
	return &PostgresKnowledgeCandidateRepository{db: db}
}

func (repository *PostgresKnowledgeCandidateRepository) Save(candidate KnowledgeCandidate) KnowledgeCandidate {
	if candidate.ID == "" {
		candidate.ID = persistence.NewID("kc")
	}

	if _, err := repository.db.Exec(
		`INSERT INTO knowledge_candidates (id, knowledge_base_id, organization_id, source_ticket_id, title, summary, content, status, created_by, reviewed_by, review_comment, approved_document_id, created_at, updated_at, reviewed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) ON CONFLICT (id) DO UPDATE SET knowledge_base_id = EXCLUDED.knowledge_base_id, organization_id = EXCLUDED.organization_id, source_ticket_id = EXCLUDED.source_ticket_id, title = EXCLUDED.title, summary = EXCLUDED.summary, content = EXCLUDED.content, status = EXCLUDED.status, created_by = EXCLUDED.created_by, reviewed_by = EXCLUDED.reviewed_by, review_comment = EXCLUDED.review_comment, approved_document_id = EXCLUDED.approved_document_id, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at, reviewed_at = EXCLUDED.reviewed_at`,
		candidate.ID,
		candidate.KnowledgeBaseID,
		candidate.OrganizationID,
		candidate.SourceTicketID,
		candidate.Title,
		candidate.Summary,
		candidate.Content,
		string(candidate.Status),
		candidate.CreatedBy,
		candidate.ReviewedBy,
		candidate.ReviewComment,
		candidate.ApprovedDocumentID,
		candidate.CreatedAt,
		candidate.UpdatedAt,
		persistence.NullableTime(candidate.ReviewedAt),
	); err != nil {
		panic(fmt.Sprintf("save knowledge candidate to postgres: %v", err))
	}

	return candidate
}

func (repository *PostgresKnowledgeCandidateRepository) FindByID(candidateID string) (KnowledgeCandidate, bool) {
	row := repository.db.QueryRow(`SELECT id, knowledge_base_id, organization_id, source_ticket_id, title, summary, content, status, created_by, reviewed_by, review_comment, approved_document_id, created_at, updated_at, reviewed_at FROM knowledge_candidates WHERE id = $1`, candidateID)
	return scanKnowledgeCandidate(row)
}

func (repository *PostgresKnowledgeCandidateRepository) ListByKnowledgeBase(knowledgeBaseID string) []KnowledgeCandidate {
	rows, err := repository.db.Query(`SELECT id, knowledge_base_id, organization_id, source_ticket_id, title, summary, content, status, created_by, reviewed_by, review_comment, approved_document_id, created_at, updated_at, reviewed_at FROM knowledge_candidates WHERE knowledge_base_id = $1 ORDER BY created_at ASC, id ASC`, knowledgeBaseID)
	if err != nil {
		panic(fmt.Sprintf("list knowledge candidates from postgres: %v", err))
	}
	defer rows.Close()

	items := make([]KnowledgeCandidate, 0)
	for rows.Next() {
		item, ok := scanKnowledgeCandidate(rows)
		if ok {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate knowledge candidates from postgres: %v", err))
	}

	return items
}

func scanKnowledgeCandidate(scanner knowledgeRowScanner) (KnowledgeCandidate, bool) {
	candidate := KnowledgeCandidate{}
	var status string
	var reviewedAt sql.NullTime
	if err := scanner.Scan(
		&candidate.ID,
		&candidate.KnowledgeBaseID,
		&candidate.OrganizationID,
		&candidate.SourceTicketID,
		&candidate.Title,
		&candidate.Summary,
		&candidate.Content,
		&status,
		&candidate.CreatedBy,
		&candidate.ReviewedBy,
		&candidate.ReviewComment,
		&candidate.ApprovedDocumentID,
		&candidate.CreatedAt,
		&candidate.UpdatedAt,
		&reviewedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return KnowledgeCandidate{}, false
		}
		panic(fmt.Sprintf("scan knowledge candidate from postgres: %v", err))
	}

	candidate.Status = KnowledgeCandidateStatus(status)
	candidate.ReviewedAt = persistence.FromNullableTime(reviewedAt)
	return candidate, true
}

package knowledge

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresKnowledgeCandidateRepositorySaveFindAndListByKnowledgeBase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresKnowledgeCandidateRepository(db)
	createdAt := time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	reviewedAt := updatedAt.Add(time.Minute)
	candidate := KnowledgeCandidate{
		KnowledgeBaseID:    "kb-1",
		OrganizationID:     "org-1",
		SourceTicketID:     "ticket-1",
		Title:              "VPN 691 错误排查",
		Summary:            "通过重置账号拨号权限恢复",
		Content:            "处理步骤：检查账号状态，重置账号拨号权限，重新连接验证。",
		Status:             KnowledgeCandidateStatusApproved,
		CreatedBy:          "user-agent-1",
		ReviewedBy:         "user-reviewer-1",
		ReviewComment:      "内容可入库",
		ApprovedDocumentID: "doc-1",
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		ReviewedAt:         reviewedAt,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO knowledge_candidates (id, knowledge_base_id, organization_id, source_ticket_id, title, summary, content, status, created_by, reviewed_by, review_comment, approved_document_id, created_at, updated_at, reviewed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) ON CONFLICT (id) DO UPDATE SET knowledge_base_id = EXCLUDED.knowledge_base_id, organization_id = EXCLUDED.organization_id, source_ticket_id = EXCLUDED.source_ticket_id, title = EXCLUDED.title, summary = EXCLUDED.summary, content = EXCLUDED.content, status = EXCLUDED.status, created_by = EXCLUDED.created_by, reviewed_by = EXCLUDED.reviewed_by, review_comment = EXCLUDED.review_comment, approved_document_id = EXCLUDED.approved_document_id, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at, reviewed_at = EXCLUDED.reviewed_at")).
		WithArgs(sqlmock.AnyArg(), candidate.KnowledgeBaseID, candidate.OrganizationID, candidate.SourceTicketID, candidate.Title, candidate.Summary, candidate.Content, string(candidate.Status), candidate.CreatedBy, candidate.ReviewedBy, candidate.ReviewComment, candidate.ApprovedDocumentID, candidate.CreatedAt, candidate.UpdatedAt, candidate.ReviewedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(candidate)
	if saved.ID == "" {
		t.Fatalf("expected generated knowledge candidate id")
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, knowledge_base_id, organization_id, source_ticket_id, title, summary, content, status, created_by, reviewed_by, review_comment, approved_document_id, created_at, updated_at, reviewed_at FROM knowledge_candidates WHERE id = $1")).
		WithArgs(saved.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "knowledge_base_id", "organization_id", "source_ticket_id", "title", "summary", "content", "status", "created_by", "reviewed_by", "review_comment", "approved_document_id", "created_at", "updated_at", "reviewed_at"}).AddRow(saved.ID, candidate.KnowledgeBaseID, candidate.OrganizationID, candidate.SourceTicketID, candidate.Title, candidate.Summary, candidate.Content, string(candidate.Status), candidate.CreatedBy, candidate.ReviewedBy, candidate.ReviewComment, candidate.ApprovedDocumentID, candidate.CreatedAt, candidate.UpdatedAt, candidate.ReviewedAt))

	stored, ok := repository.FindByID(saved.ID)
	if !ok {
		t.Fatalf("expected knowledge candidate to be found")
	}
	if stored.ApprovedDocumentID != candidate.ApprovedDocumentID {
		t.Fatalf("expected approved document id %q, got %q", candidate.ApprovedDocumentID, stored.ApprovedDocumentID)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, knowledge_base_id, organization_id, source_ticket_id, title, summary, content, status, created_by, reviewed_by, review_comment, approved_document_id, created_at, updated_at, reviewed_at FROM knowledge_candidates WHERE knowledge_base_id = $1 ORDER BY created_at ASC, id ASC")).
		WithArgs(candidate.KnowledgeBaseID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "knowledge_base_id", "organization_id", "source_ticket_id", "title", "summary", "content", "status", "created_by", "reviewed_by", "review_comment", "approved_document_id", "created_at", "updated_at", "reviewed_at"}).AddRow(saved.ID, candidate.KnowledgeBaseID, candidate.OrganizationID, candidate.SourceTicketID, candidate.Title, candidate.Summary, candidate.Content, string(candidate.Status), candidate.CreatedBy, candidate.ReviewedBy, candidate.ReviewComment, candidate.ApprovedDocumentID, candidate.CreatedAt, candidate.UpdatedAt, candidate.ReviewedAt))

	items := repository.ListByKnowledgeBase(candidate.KnowledgeBaseID)
	if len(items) != 1 {
		t.Fatalf("expected 1 knowledge candidate, got %d", len(items))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

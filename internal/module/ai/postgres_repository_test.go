package ai

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresUnifiedIntakeRepositorySaveAndFindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresUnifiedIntakeRepository(db)
	createdAt := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	record := UnifiedIntakeRecord{
		OrganizationID:  "org-1",
		RequesterID:     "user-1",
		KnowledgeBaseID: "kb-1",
		Question:        "VPN 无法连接怎么办？",
		ResultType:      UnifiedIntakeResultTypeAnswered,
		AnswerStatus:    AnswerStatusAnswered,
		Answer:          "请先重置 VPN 客户端。",
		Confidence:      0.82,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt.Add(time.Minute),
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ai_unified_intakes (id, organization_id, requester_id, knowledge_base_id, question, result_type, answer_status, answer, confidence, ticket_id, escalated_ticket_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT (id) DO UPDATE SET organization_id = EXCLUDED.organization_id, requester_id = EXCLUDED.requester_id, knowledge_base_id = EXCLUDED.knowledge_base_id, question = EXCLUDED.question, result_type = EXCLUDED.result_type, answer_status = EXCLUDED.answer_status, answer = EXCLUDED.answer, confidence = EXCLUDED.confidence, ticket_id = EXCLUDED.ticket_id, escalated_ticket_id = EXCLUDED.escalated_ticket_id, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at")).
		WithArgs(sqlmock.AnyArg(), record.OrganizationID, record.RequesterID, record.KnowledgeBaseID, record.Question, string(record.ResultType), string(record.AnswerStatus), record.Answer, record.Confidence, record.TicketID, record.EscalatedTicketID, record.CreatedAt, record.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(record)
	if saved.ID == "" {
		t.Fatalf("expected generated intake id")
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, organization_id, requester_id, knowledge_base_id, question, result_type, answer_status, answer, confidence, ticket_id, escalated_ticket_id, created_at, updated_at FROM ai_unified_intakes WHERE id = $1")).
		WithArgs(saved.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "requester_id", "knowledge_base_id", "question", "result_type", "answer_status", "answer", "confidence", "ticket_id", "escalated_ticket_id", "created_at", "updated_at"}).
			AddRow(saved.ID, record.OrganizationID, record.RequesterID, record.KnowledgeBaseID, record.Question, string(record.ResultType), string(record.AnswerStatus), record.Answer, record.Confidence, record.TicketID, record.EscalatedTicketID, record.CreatedAt, record.UpdatedAt))

	stored, ok := repository.FindByID(saved.ID)
	if !ok {
		t.Fatalf("expected intake to be found")
	}
	if stored.Question != record.Question {
		t.Fatalf("expected question %q, got %q", record.Question, stored.Question)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPostgresAnswerFeedbackRepositorySaveFindAndList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresAnswerFeedbackRepository(db)
	createdAt := time.Date(2026, 3, 9, 13, 0, 0, 0, time.UTC)
	feedback := AnswerFeedback{
		IntakeID:       "intake-1",
		OrganizationID: "org-1",
		ActorID:        "user-1",
		Status:         AnswerFeedbackStatusUnresolved,
		Comment:        "仍未解决",
		TicketAction:   AnswerFeedbackTicketActionTicketCreated,
		TicketID:       "ticket-2",
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt.Add(time.Minute),
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ai_answer_feedbacks (id, intake_id, organization_id, actor_id, status, comment, ticket_action, ticket_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (id) DO UPDATE SET intake_id = EXCLUDED.intake_id, organization_id = EXCLUDED.organization_id, actor_id = EXCLUDED.actor_id, status = EXCLUDED.status, comment = EXCLUDED.comment, ticket_action = EXCLUDED.ticket_action, ticket_id = EXCLUDED.ticket_id, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at")).
		WithArgs(sqlmock.AnyArg(), feedback.IntakeID, feedback.OrganizationID, feedback.ActorID, string(feedback.Status), feedback.Comment, string(feedback.TicketAction), feedback.TicketID, feedback.CreatedAt, feedback.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(feedback)
	if saved.ID == "" {
		t.Fatalf("expected generated feedback id")
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, intake_id, organization_id, actor_id, status, comment, ticket_action, ticket_id, created_at, updated_at FROM ai_answer_feedbacks WHERE intake_id = $1 AND actor_id = $2")).
		WithArgs(feedback.IntakeID, feedback.ActorID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "intake_id", "organization_id", "actor_id", "status", "comment", "ticket_action", "ticket_id", "created_at", "updated_at"}).
			AddRow(saved.ID, feedback.IntakeID, feedback.OrganizationID, feedback.ActorID, string(feedback.Status), feedback.Comment, string(feedback.TicketAction), feedback.TicketID, feedback.CreatedAt, feedback.UpdatedAt))

	stored, ok := repository.FindByIntakeAndActor(feedback.IntakeID, feedback.ActorID)
	if !ok {
		t.Fatalf("expected feedback to be found")
	}
	if stored.TicketID != feedback.TicketID {
		t.Fatalf("expected ticket id %q, got %q", feedback.TicketID, stored.TicketID)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, intake_id, organization_id, actor_id, status, comment, ticket_action, ticket_id, created_at, updated_at FROM ai_answer_feedbacks WHERE organization_id = $1 ORDER BY created_at ASC, id ASC")).
		WithArgs(feedback.OrganizationID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "intake_id", "organization_id", "actor_id", "status", "comment", "ticket_action", "ticket_id", "created_at", "updated_at"}).
			AddRow(saved.ID, feedback.IntakeID, feedback.OrganizationID, feedback.ActorID, string(feedback.Status), feedback.Comment, string(feedback.TicketAction), feedback.TicketID, feedback.CreatedAt, feedback.UpdatedAt))

	items := repository.ListByOrganization(feedback.OrganizationID)
	if len(items) != 1 {
		t.Fatalf("expected 1 feedback, got %d", len(items))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

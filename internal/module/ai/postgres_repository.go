package ai

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/persistence"
)

// PostgresUnifiedIntakeRepository 使用 PostgreSQL 持久化统一受理主记录。
type PostgresUnifiedIntakeRepository struct {
	db *sql.DB
}

func NewPostgresUnifiedIntakeRepository(db *sql.DB) *PostgresUnifiedIntakeRepository {
	return &PostgresUnifiedIntakeRepository{db: db}
}

func (repository *PostgresUnifiedIntakeRepository) Save(record UnifiedIntakeRecord) UnifiedIntakeRecord {
	if record.ID == "" {
		record.ID = persistence.NewID("intake")
	}

	if _, err := repository.db.Exec(
		`INSERT INTO ai_unified_intakes (id, organization_id, requester_id, knowledge_base_id, question, result_type, answer_status, answer, confidence, ticket_id, escalated_ticket_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT (id) DO UPDATE SET organization_id = EXCLUDED.organization_id, requester_id = EXCLUDED.requester_id, knowledge_base_id = EXCLUDED.knowledge_base_id, question = EXCLUDED.question, result_type = EXCLUDED.result_type, answer_status = EXCLUDED.answer_status, answer = EXCLUDED.answer, confidence = EXCLUDED.confidence, ticket_id = EXCLUDED.ticket_id, escalated_ticket_id = EXCLUDED.escalated_ticket_id, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at`,
		record.ID,
		record.OrganizationID,
		record.RequesterID,
		record.KnowledgeBaseID,
		record.Question,
		string(record.ResultType),
		string(record.AnswerStatus),
		record.Answer,
		record.Confidence,
		record.TicketID,
		record.EscalatedTicketID,
		record.CreatedAt,
		record.UpdatedAt,
	); err != nil {
		panic(fmt.Sprintf("save unified intake to postgres: %v", err))
	}

	return record
}

func (repository *PostgresUnifiedIntakeRepository) FindByID(intakeID string) (UnifiedIntakeRecord, bool) {
	row := repository.db.QueryRow(`SELECT id, organization_id, requester_id, knowledge_base_id, question, result_type, answer_status, answer, confidence, ticket_id, escalated_ticket_id, created_at, updated_at FROM ai_unified_intakes WHERE id = $1`, intakeID)
	return scanUnifiedIntakeRecord(row)
}

// PostgresAnswerFeedbackRepository 使用 PostgreSQL 持久化反馈记录。
type PostgresAnswerFeedbackRepository struct {
	db *sql.DB
}

func NewPostgresAnswerFeedbackRepository(db *sql.DB) *PostgresAnswerFeedbackRepository {
	return &PostgresAnswerFeedbackRepository{db: db}
}

func (repository *PostgresAnswerFeedbackRepository) Save(feedback AnswerFeedback) AnswerFeedback {
	if feedback.ID == "" {
		feedback.ID = persistence.NewID("feedback")
	}

	if _, err := repository.db.Exec(
		`INSERT INTO ai_answer_feedbacks (id, intake_id, organization_id, actor_id, status, comment, ticket_action, ticket_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (id) DO UPDATE SET intake_id = EXCLUDED.intake_id, organization_id = EXCLUDED.organization_id, actor_id = EXCLUDED.actor_id, status = EXCLUDED.status, comment = EXCLUDED.comment, ticket_action = EXCLUDED.ticket_action, ticket_id = EXCLUDED.ticket_id, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at`,
		feedback.ID,
		feedback.IntakeID,
		feedback.OrganizationID,
		feedback.ActorID,
		string(feedback.Status),
		feedback.Comment,
		string(feedback.TicketAction),
		feedback.TicketID,
		feedback.CreatedAt,
		feedback.UpdatedAt,
	); err != nil {
		panic(fmt.Sprintf("save answer feedback to postgres: %v", err))
	}

	return feedback
}

func (repository *PostgresAnswerFeedbackRepository) FindByIntakeAndActor(intakeID, actorID string) (AnswerFeedback, bool) {
	row := repository.db.QueryRow(`SELECT id, intake_id, organization_id, actor_id, status, comment, ticket_action, ticket_id, created_at, updated_at FROM ai_answer_feedbacks WHERE intake_id = $1 AND actor_id = $2`, intakeID, actorID)
	return scanAnswerFeedback(row)
}

func (repository *PostgresAnswerFeedbackRepository) ListByOrganization(organizationID string) []AnswerFeedback {
	rows, err := repository.db.Query(`SELECT id, intake_id, organization_id, actor_id, status, comment, ticket_action, ticket_id, created_at, updated_at FROM ai_answer_feedbacks WHERE organization_id = $1 ORDER BY created_at ASC, id ASC`, organizationID)
	if err != nil {
		panic(fmt.Sprintf("list answer feedbacks from postgres: %v", err))
	}
	defer rows.Close()

	items := make([]AnswerFeedback, 0)
	for rows.Next() {
		item, ok := scanAnswerFeedback(rows)
		if ok {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate answer feedbacks from postgres: %v", err))
	}

	return items
}

type aiRowScanner interface {
	Scan(dest ...any) error
}

func scanUnifiedIntakeRecord(scanner aiRowScanner) (UnifiedIntakeRecord, bool) {
	record := UnifiedIntakeRecord{}
	var resultType string
	var answerStatus string
	if err := scanner.Scan(&record.ID, &record.OrganizationID, &record.RequesterID, &record.KnowledgeBaseID, &record.Question, &resultType, &answerStatus, &record.Answer, &record.Confidence, &record.TicketID, &record.EscalatedTicketID, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UnifiedIntakeRecord{}, false
		}
		panic(fmt.Sprintf("scan unified intake from postgres: %v", err))
	}
	record.ResultType = UnifiedIntakeResultType(resultType)
	record.AnswerStatus = AnswerStatus(answerStatus)
	return record, true
}

func scanAnswerFeedback(scanner aiRowScanner) (AnswerFeedback, bool) {
	feedback := AnswerFeedback{}
	var status string
	var ticketAction string
	if err := scanner.Scan(&feedback.ID, &feedback.IntakeID, &feedback.OrganizationID, &feedback.ActorID, &status, &feedback.Comment, &ticketAction, &feedback.TicketID, &feedback.CreatedAt, &feedback.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AnswerFeedback{}, false
		}
		panic(fmt.Sprintf("scan answer feedback from postgres: %v", err))
	}
	feedback.Status = AnswerFeedbackStatus(status)
	feedback.TicketAction = AnswerFeedbackTicketAction(ticketAction)
	return feedback, true
}

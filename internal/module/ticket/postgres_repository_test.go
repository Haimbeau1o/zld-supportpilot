package ticket

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresTicketRepositorySaveFindAndListByOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresTicketRepository(db)
	createdAt := time.Date(2026, 3, 9, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	ticket := Ticket{
		OrganizationID: "org-1",
		RequesterID:    "user-end-1",
		AssigneeID:     "user-agent-1",
		Title:          "VPN 无法连接",
		Description:    "今天上午开始无法连接公司 VPN",
		Category:       "network",
		Priority:       TicketPriorityHigh,
		Status:         TicketStatusOpen,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO tickets (id, organization_id, requester_id, assignee_id, title, description, category, priority, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (id) DO UPDATE SET organization_id = EXCLUDED.organization_id, requester_id = EXCLUDED.requester_id, assignee_id = EXCLUDED.assignee_id, title = EXCLUDED.title, description = EXCLUDED.description, category = EXCLUDED.category, priority = EXCLUDED.priority, status = EXCLUDED.status, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at")).
		WithArgs(sqlmock.AnyArg(), ticket.OrganizationID, ticket.RequesterID, ticket.AssigneeID, ticket.Title, ticket.Description, ticket.Category, string(ticket.Priority), string(ticket.Status), ticket.CreatedAt, ticket.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(ticket)
	if saved.ID == "" {
		t.Fatalf("expected generated ticket id")
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, organization_id, requester_id, assignee_id, title, description, category, priority, status, created_at, updated_at FROM tickets WHERE id = $1")).
		WithArgs(saved.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "requester_id", "assignee_id", "title", "description", "category", "priority", "status", "created_at", "updated_at"}).AddRow(saved.ID, ticket.OrganizationID, ticket.RequesterID, ticket.AssigneeID, ticket.Title, ticket.Description, ticket.Category, string(ticket.Priority), string(ticket.Status), ticket.CreatedAt, ticket.UpdatedAt))

	stored, ok := repository.FindByID(saved.ID)
	if !ok {
		t.Fatalf("expected stored ticket to be found")
	}
	if stored.Title != ticket.Title {
		t.Fatalf("expected title %q, got %q", ticket.Title, stored.Title)
	}

	secondTicketID := "ticket-2"
	secondCreatedAt := createdAt.Add(time.Hour)
	secondUpdatedAt := secondCreatedAt.Add(time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, organization_id, requester_id, assignee_id, title, description, category, priority, status, created_at, updated_at FROM tickets WHERE organization_id = $1 ORDER BY created_at ASC, id ASC")).
		WithArgs(ticket.OrganizationID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "requester_id", "assignee_id", "title", "description", "category", "priority", "status", "created_at", "updated_at"}).
			AddRow(saved.ID, ticket.OrganizationID, ticket.RequesterID, ticket.AssigneeID, ticket.Title, ticket.Description, ticket.Category, string(ticket.Priority), string(ticket.Status), ticket.CreatedAt, ticket.UpdatedAt).
			AddRow(secondTicketID, ticket.OrganizationID, ticket.RequesterID, ticket.AssigneeID, "邮箱无法发送", "SMTP 发送失败", "mail", string(TicketPriorityMedium), string(TicketStatusInProgress), secondCreatedAt, secondUpdatedAt))

	items := repository.ListByOrganization(ticket.OrganizationID)
	if len(items) != 2 {
		t.Fatalf("expected 2 tickets, got %d", len(items))
	}
	if items[0].ID != saved.ID || items[1].ID != secondTicketID {
		t.Fatalf("expected ordered tickets [%q %q], got [%q %q]", saved.ID, secondTicketID, items[0].ID, items[1].ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPostgresTicketCommentRepositorySaveAndListByTicket(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresTicketCommentRepository(db)
	createdAt := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	comment := TicketComment{
		TicketID:       "ticket-1",
		OrganizationID: "org-1",
		AuthorID:       "user-agent-1",
		Type:           TicketCommentTypeInternalNote,
		Content:        "已联系网络组排查",
		CreatedAt:      createdAt,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ticket_comments (id, ticket_id, organization_id, author_id, type, content, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO UPDATE SET ticket_id = EXCLUDED.ticket_id, organization_id = EXCLUDED.organization_id, author_id = EXCLUDED.author_id, type = EXCLUDED.type, content = EXCLUDED.content, created_at = EXCLUDED.created_at")).
		WithArgs(sqlmock.AnyArg(), comment.TicketID, comment.OrganizationID, comment.AuthorID, string(comment.Type), comment.Content, comment.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(comment)
	if saved.ID == "" {
		t.Fatalf("expected generated comment id")
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, ticket_id, organization_id, author_id, type, content, created_at FROM ticket_comments WHERE ticket_id = $1 ORDER BY created_at ASC, id ASC")).
		WithArgs(comment.TicketID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "ticket_id", "organization_id", "author_id", "type", "content", "created_at"}).AddRow(saved.ID, comment.TicketID, comment.OrganizationID, comment.AuthorID, string(comment.Type), comment.Content, comment.CreatedAt))

	items := repository.ListByTicket(comment.TicketID)
	if len(items) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(items))
	}
	if items[0].Content != comment.Content {
		t.Fatalf("expected content %q, got %q", comment.Content, items[0].Content)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPostgresTicketAuditEventRepositorySaveAndListByTicket(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresTicketAuditEventRepository(db)
	createdAt := time.Date(2026, 3, 9, 11, 0, 0, 0, time.UTC)
	event := TicketAuditEvent{
		TicketID:       "ticket-1",
		OrganizationID: "org-1",
		ActorID:        "user-agent-1",
		Type:           TicketAuditEventTypeStatusChanged,
		Content:        "状态由 open 变更为 in_progress",
		FromValue:      string(TicketStatusOpen),
		ToValue:        string(TicketStatusInProgress),
		CreatedAt:      createdAt,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ticket_audit_events (id, ticket_id, organization_id, actor_id, type, content, from_value, to_value, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (id) DO UPDATE SET ticket_id = EXCLUDED.ticket_id, organization_id = EXCLUDED.organization_id, actor_id = EXCLUDED.actor_id, type = EXCLUDED.type, content = EXCLUDED.content, from_value = EXCLUDED.from_value, to_value = EXCLUDED.to_value, created_at = EXCLUDED.created_at")).
		WithArgs(sqlmock.AnyArg(), event.TicketID, event.OrganizationID, event.ActorID, string(event.Type), event.Content, event.FromValue, event.ToValue, event.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(event)
	if saved.ID == "" {
		t.Fatalf("expected generated audit event id")
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, ticket_id, organization_id, actor_id, type, content, from_value, to_value, created_at FROM ticket_audit_events WHERE ticket_id = $1 ORDER BY created_at ASC, id ASC")).
		WithArgs(event.TicketID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "ticket_id", "organization_id", "actor_id", "type", "content", "from_value", "to_value", "created_at"}).AddRow(saved.ID, event.TicketID, event.OrganizationID, event.ActorID, string(event.Type), event.Content, event.FromValue, event.ToValue, event.CreatedAt))

	items := repository.ListByTicket(event.TicketID)
	if len(items) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(items))
	}
	if items[0].ToValue != event.ToValue {
		t.Fatalf("expected to value %q, got %q", event.ToValue, items[0].ToValue)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

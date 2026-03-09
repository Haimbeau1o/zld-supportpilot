package ticket

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/persistence"
)

// PostgresTicketRepository 使用 PostgreSQL 持久化工单主模型。
type PostgresTicketRepository struct {
	db *sql.DB
}

func NewPostgresTicketRepository(db *sql.DB) *PostgresTicketRepository {
	return &PostgresTicketRepository{db: db}
}

func (repository *PostgresTicketRepository) Save(ticket Ticket) Ticket {
	if ticket.ID == "" {
		ticket.ID = persistence.NewID("ticket")
	}

	if _, err := repository.db.Exec(
		`INSERT INTO tickets (id, organization_id, requester_id, assignee_id, title, description, category, priority, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (id) DO UPDATE SET organization_id = EXCLUDED.organization_id, requester_id = EXCLUDED.requester_id, assignee_id = EXCLUDED.assignee_id, title = EXCLUDED.title, description = EXCLUDED.description, category = EXCLUDED.category, priority = EXCLUDED.priority, status = EXCLUDED.status, created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at`,
		ticket.ID,
		ticket.OrganizationID,
		ticket.RequesterID,
		ticket.AssigneeID,
		ticket.Title,
		ticket.Description,
		ticket.Category,
		string(ticket.Priority),
		string(ticket.Status),
		ticket.CreatedAt,
		ticket.UpdatedAt,
	); err != nil {
		panic(fmt.Sprintf("save ticket to postgres: %v", err))
	}

	return ticket
}

func (repository *PostgresTicketRepository) FindByID(ticketID string) (Ticket, bool) {
	row := repository.db.QueryRow(`SELECT id, organization_id, requester_id, assignee_id, title, description, category, priority, status, created_at, updated_at FROM tickets WHERE id = $1`, ticketID)
	return scanTicket(row)
}

func (repository *PostgresTicketRepository) ListByOrganization(organizationID string) []Ticket {
	rows, err := repository.db.Query(`SELECT id, organization_id, requester_id, assignee_id, title, description, category, priority, status, created_at, updated_at FROM tickets WHERE organization_id = $1 ORDER BY created_at ASC, id ASC`, organizationID)
	if err != nil {
		panic(fmt.Sprintf("list tickets by organization from postgres: %v", err))
	}
	defer rows.Close()

	tickets := make([]Ticket, 0)
	for rows.Next() {
		ticket, ok := scanTicket(rows)
		if ok {
			tickets = append(tickets, ticket)
		}
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate tickets from postgres: %v", err))
	}

	return tickets
}

// PostgresTicketCommentRepository 使用 PostgreSQL 持久化评论与备注。
type PostgresTicketCommentRepository struct {
	db *sql.DB
}

func NewPostgresTicketCommentRepository(db *sql.DB) *PostgresTicketCommentRepository {
	return &PostgresTicketCommentRepository{db: db}
}

func (repository *PostgresTicketCommentRepository) Save(comment TicketComment) TicketComment {
	if comment.ID == "" {
		comment.ID = persistence.NewID("comment")
	}

	if _, err := repository.db.Exec(
		`INSERT INTO ticket_comments (id, ticket_id, organization_id, author_id, type, content, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO UPDATE SET ticket_id = EXCLUDED.ticket_id, organization_id = EXCLUDED.organization_id, author_id = EXCLUDED.author_id, type = EXCLUDED.type, content = EXCLUDED.content, created_at = EXCLUDED.created_at`,
		comment.ID,
		comment.TicketID,
		comment.OrganizationID,
		comment.AuthorID,
		string(comment.Type),
		comment.Content,
		comment.CreatedAt,
	); err != nil {
		panic(fmt.Sprintf("save ticket comment to postgres: %v", err))
	}

	return comment
}

func (repository *PostgresTicketCommentRepository) ListByTicket(ticketID string) []TicketComment {
	rows, err := repository.db.Query(`SELECT id, ticket_id, organization_id, author_id, type, content, created_at FROM ticket_comments WHERE ticket_id = $1 ORDER BY created_at ASC, id ASC`, ticketID)
	if err != nil {
		panic(fmt.Sprintf("list ticket comments from postgres: %v", err))
	}
	defer rows.Close()

	comments := make([]TicketComment, 0)
	for rows.Next() {
		comment := TicketComment{}
		var commentType string
		if err := rows.Scan(&comment.ID, &comment.TicketID, &comment.OrganizationID, &comment.AuthorID, &commentType, &comment.Content, &comment.CreatedAt); err != nil {
			panic(fmt.Sprintf("scan ticket comment from postgres: %v", err))
		}
		comment.Type = TicketCommentType(commentType)
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate ticket comments from postgres: %v", err))
	}

	return comments
}

// PostgresTicketAuditEventRepository 使用 PostgreSQL 持久化审计事件。
type PostgresTicketAuditEventRepository struct {
	db *sql.DB
}

func NewPostgresTicketAuditEventRepository(db *sql.DB) *PostgresTicketAuditEventRepository {
	return &PostgresTicketAuditEventRepository{db: db}
}

func (repository *PostgresTicketAuditEventRepository) Save(event TicketAuditEvent) TicketAuditEvent {
	if event.ID == "" {
		event.ID = persistence.NewID("audit")
	}

	if _, err := repository.db.Exec(
		`INSERT INTO ticket_audit_events (id, ticket_id, organization_id, actor_id, type, content, from_value, to_value, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (id) DO UPDATE SET ticket_id = EXCLUDED.ticket_id, organization_id = EXCLUDED.organization_id, actor_id = EXCLUDED.actor_id, type = EXCLUDED.type, content = EXCLUDED.content, from_value = EXCLUDED.from_value, to_value = EXCLUDED.to_value, created_at = EXCLUDED.created_at`,
		event.ID,
		event.TicketID,
		event.OrganizationID,
		event.ActorID,
		string(event.Type),
		event.Content,
		event.FromValue,
		event.ToValue,
		event.CreatedAt,
	); err != nil {
		panic(fmt.Sprintf("save ticket audit event to postgres: %v", err))
	}

	return event
}

func (repository *PostgresTicketAuditEventRepository) ListByTicket(ticketID string) []TicketAuditEvent {
	rows, err := repository.db.Query(`SELECT id, ticket_id, organization_id, actor_id, type, content, from_value, to_value, created_at FROM ticket_audit_events WHERE ticket_id = $1 ORDER BY created_at ASC, id ASC`, ticketID)
	if err != nil {
		panic(fmt.Sprintf("list ticket audit events from postgres: %v", err))
	}
	defer rows.Close()

	events := make([]TicketAuditEvent, 0)
	for rows.Next() {
		event := TicketAuditEvent{}
		var eventType string
		if err := rows.Scan(&event.ID, &event.TicketID, &event.OrganizationID, &event.ActorID, &eventType, &event.Content, &event.FromValue, &event.ToValue, &event.CreatedAt); err != nil {
			panic(fmt.Sprintf("scan ticket audit event from postgres: %v", err))
		}
		event.Type = TicketAuditEventType(eventType)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("iterate ticket audit events from postgres: %v", err))
	}

	return events
}

type ticketRowScanner interface {
	Scan(dest ...any) error
}

func scanTicket(scanner ticketRowScanner) (Ticket, bool) {
	ticket := Ticket{}
	var priority string
	var status string
	if err := scanner.Scan(&ticket.ID, &ticket.OrganizationID, &ticket.RequesterID, &ticket.AssigneeID, &ticket.Title, &ticket.Description, &ticket.Category, &priority, &status, &ticket.CreatedAt, &ticket.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Ticket{}, false
		}
		panic(fmt.Sprintf("scan ticket from postgres: %v", err))
	}
	ticket.Priority = TicketPriority(priority)
	ticket.Status = TicketStatus(status)
	return ticket, true
}

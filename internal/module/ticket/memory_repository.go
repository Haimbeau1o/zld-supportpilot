package ticket

import (
	"fmt"
	"sort"
	"sync"
)

// MemoryTicketRepository 使用内存维护工单，当前阶段用于先稳定主流程边界和测试行为。
type MemoryTicketRepository struct {
	mutex         sync.RWMutex
	ticketsByID   map[string]Ticket
	nextTicketSeq uint64
}

func NewMemoryTicketRepository() *MemoryTicketRepository {
	return &MemoryTicketRepository{
		ticketsByID: make(map[string]Ticket),
	}
}

func (repository *MemoryTicketRepository) Save(ticket Ticket) Ticket {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if ticket.ID == "" {
		repository.nextTicketSeq++
		ticket.ID = fmt.Sprintf("ticket-%d", repository.nextTicketSeq)
	}

	repository.ticketsByID[ticket.ID] = ticket
	return ticket
}

func (repository *MemoryTicketRepository) FindByID(ticketID string) (Ticket, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	ticket, ok := repository.ticketsByID[ticketID]
	return ticket, ok
}

func (repository *MemoryTicketRepository) ListByOrganization(organizationID string) []Ticket {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	tickets := make([]Ticket, 0)
	for _, ticket := range repository.ticketsByID {
		if ticket.OrganizationID == organizationID {
			tickets = append(tickets, ticket)
		}
	}

	sort.Slice(tickets, func(left, right int) bool {
		if tickets[left].CreatedAt.Equal(tickets[right].CreatedAt) {
			return tickets[left].ID < tickets[right].ID
		}

		return tickets[left].CreatedAt.Before(tickets[right].CreatedAt)
	})

	return tickets
}

// MemoryTicketCommentRepository 使用内存维护评论与内部备注。
type MemoryTicketCommentRepository struct {
	mutex          sync.RWMutex
	commentsByID   map[string]TicketComment
	nextCommentSeq uint64
}

func NewMemoryTicketCommentRepository() *MemoryTicketCommentRepository {
	return &MemoryTicketCommentRepository{commentsByID: make(map[string]TicketComment)}
}

func (repository *MemoryTicketCommentRepository) Save(comment TicketComment) TicketComment {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if comment.ID == "" {
		repository.nextCommentSeq++
		comment.ID = fmt.Sprintf("comment-%d", repository.nextCommentSeq)
	}

	repository.commentsByID[comment.ID] = comment
	return comment
}

func (repository *MemoryTicketCommentRepository) ListByTicket(ticketID string) []TicketComment {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	comments := make([]TicketComment, 0)
	for _, comment := range repository.commentsByID {
		if comment.TicketID == ticketID {
			comments = append(comments, comment)
		}
	}

	sort.Slice(comments, func(left, right int) bool {
		if comments[left].CreatedAt.Equal(comments[right].CreatedAt) {
			return comments[left].ID < comments[right].ID
		}
		return comments[left].CreatedAt.Before(comments[right].CreatedAt)
	})

	return comments
}

// MemoryTicketAuditEventRepository 使用内存维护审计事件。
type MemoryTicketAuditEventRepository struct {
	mutex        sync.RWMutex
	eventsByID   map[string]TicketAuditEvent
	nextEventSeq uint64
}

func NewMemoryTicketAuditEventRepository() *MemoryTicketAuditEventRepository {
	return &MemoryTicketAuditEventRepository{eventsByID: make(map[string]TicketAuditEvent)}
}

func (repository *MemoryTicketAuditEventRepository) Save(event TicketAuditEvent) TicketAuditEvent {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if event.ID == "" {
		repository.nextEventSeq++
		event.ID = fmt.Sprintf("audit-%d", repository.nextEventSeq)
	}

	repository.eventsByID[event.ID] = event
	return event
}

func (repository *MemoryTicketAuditEventRepository) ListByTicket(ticketID string) []TicketAuditEvent {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	events := make([]TicketAuditEvent, 0)
	for _, event := range repository.eventsByID {
		if event.TicketID == ticketID {
			events = append(events, event)
		}
	}

	sort.Slice(events, func(left, right int) bool {
		if events[left].CreatedAt.Equal(events[right].CreatedAt) {
			return events[left].ID < events[right].ID
		}
		return events[left].CreatedAt.Before(events[right].CreatedAt)
	})

	return events
}

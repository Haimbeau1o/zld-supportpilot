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

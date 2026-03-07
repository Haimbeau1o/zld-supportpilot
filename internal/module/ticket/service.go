package ticket

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

var (
	ErrTicketNotFound          = errors.New("ticket not found")
	ErrTicketForbidden         = errors.New("ticket forbidden")
	ErrInvalidTicketInput      = errors.New("invalid ticket input")
	ErrInvalidStatusTransition = errors.New("invalid ticket status transition")
)

type CreateTicketInput struct {
	Title       string
	Description string
	Category    string
	Priority    TicketPriority
}

type ListTicketsInput struct{}

type AssignTicketInput struct {
	TicketID   string
	AssigneeID string
}

type TransitionTicketStatusInput struct {
	TicketID string
	ToStatus TicketStatus
}

type Service struct {
	ticketRepository TicketRepository
}

func NewService(ticketRepository TicketRepository) *Service {
	return &Service{ticketRepository: ticketRepository}
}

func (service *Service) CreateTicket(actor identity.IdentityContext, input CreateTicketInput) (Ticket, error) {
	if !canCreateTicket(actor) {
		return Ticket{}, ErrTicketForbidden
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Ticket{}, fmt.Errorf("%w: title is required", ErrInvalidTicketInput)
	}

	now := time.Now()
	priority := input.Priority
	if priority == "" {
		priority = TicketPriorityMedium
	}

	ticket := service.ticketRepository.Save(Ticket{
		OrganizationID: actor.OrganizationID,
		RequesterID:    actor.UserID,
		Title:          title,
		Description:    strings.TrimSpace(input.Description),
		Category:       strings.TrimSpace(input.Category),
		Priority:       priority,
		Status:         InitialTicketStatus(),
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	return ticket, nil
}

func (service *Service) GetTicket(actor identity.IdentityContext, ticketID string) (Ticket, error) {
	ticket, ok := service.ticketRepository.FindByID(ticketID)
	if !ok {
		return Ticket{}, ErrTicketNotFound
	}

	if !canReadTicket(actor, ticket) {
		return Ticket{}, ErrTicketForbidden
	}

	return ticket, nil
}

func (service *Service) ListTickets(actor identity.IdentityContext, _ ListTicketsInput) ([]Ticket, error) {
	organizationTickets := service.ticketRepository.ListByOrganization(actor.OrganizationID)
	accessibleTickets := make([]Ticket, 0, len(organizationTickets))
	for _, ticket := range organizationTickets {
		// 查询范围必须落在服务层，而不是只写在 HTTP 层，否则后续其他入口会出现权限漂移。
		if canReadTicket(actor, ticket) {
			accessibleTickets = append(accessibleTickets, ticket)
		}
	}

	return accessibleTickets, nil
}

func (service *Service) AssignTicket(actor identity.IdentityContext, input AssignTicketInput) (Ticket, error) {
	if strings.TrimSpace(input.AssigneeID) == "" {
		return Ticket{}, fmt.Errorf("%w: assignee id is required", ErrInvalidTicketInput)
	}

	ticket, err := service.GetTicket(actor, input.TicketID)
	if err != nil {
		return Ticket{}, err
	}

	if !canManageTicket(actor, ticket) {
		return Ticket{}, ErrTicketForbidden
	}

	ticket.AssigneeID = strings.TrimSpace(input.AssigneeID)
	ticket.UpdatedAt = time.Now()
	ticket = service.ticketRepository.Save(ticket)
	return ticket, nil
}

func (service *Service) TransitionTicketStatus(actor identity.IdentityContext, input TransitionTicketStatusInput) (Ticket, error) {
	ticket, err := service.GetTicket(actor, input.TicketID)
	if err != nil {
		return Ticket{}, err
	}

	if !canManageTicket(actor, ticket) {
		return Ticket{}, ErrTicketForbidden
	}

	if err := ValidateStatusTransition(ticket.Status, input.ToStatus); err != nil {
		return Ticket{}, fmt.Errorf("%w: %v", ErrInvalidStatusTransition, err)
	}

	ticket.Status = input.ToStatus
	ticket.UpdatedAt = time.Now()
	ticket = service.ticketRepository.Save(ticket)
	return ticket, nil
}

func canCreateTicket(actor identity.IdentityContext) bool {
	return actor.Permissions.Contains(identity.PermissionTicketSelfCreate) || actor.Permissions.Contains(identity.PermissionTicketWrite)
}

func canReadTicket(actor identity.IdentityContext, ticket Ticket) bool {
	if actor.OrganizationID != ticket.OrganizationID {
		return false
	}

	if actor.Permissions.Contains(identity.PermissionTicketRead) {
		return true
	}

	// 终端用户只允许访问自己创建的工单，这样后续开放自助入口时不会天然越权看到租户内全部工单。
	return actor.Permissions.Contains(identity.PermissionTicketSelfRead) && ticket.RequesterID == actor.UserID
}

func canManageTicket(actor identity.IdentityContext, ticket Ticket) bool {
	if actor.OrganizationID != ticket.OrganizationID {
		return false
	}

	// 指派和状态流转当前统一收敛为“工单写权限”，先把主动作边界稳定下来，后续 #4 再细分评论和审计动作。
	return actor.Permissions.Contains(identity.PermissionTicketWrite)
}

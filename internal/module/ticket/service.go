package ticket

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/notify"
)

var (
	ErrTicketNotFound              = errors.New("ticket not found")
	ErrTicketForbidden             = errors.New("ticket forbidden")
	ErrInvalidTicketInput          = errors.New("invalid ticket input")
	ErrInvalidStatusTransition     = errors.New("invalid ticket status transition")
	ErrTicketCollaborationDisabled = errors.New("ticket collaboration disabled")
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

type AddTicketCommentInput struct {
	TicketID string
	Type     TicketCommentType
	Content  string
}

type CollaborationDependencies struct {
	CommentRepository TicketCommentRepository
	AuditRepository   TicketAuditEventRepository
}

type ServiceOption func(*Service)

type NotificationPublisher interface {
	Publish(ctx context.Context, event notify.Event) error
}

type Service struct {
	ticketRepository      TicketRepository
	commentRepository     TicketCommentRepository
	auditRepository       TicketAuditEventRepository
	notificationPublisher NotificationPublisher
}

func WithCollaborationDependencies(dependencies CollaborationDependencies) ServiceOption {
	return func(service *Service) {
		service.commentRepository = dependencies.CommentRepository
		service.auditRepository = dependencies.AuditRepository
	}
}

func WithNotificationPublisher(publisher NotificationPublisher) ServiceOption {
	return func(service *Service) {
		service.notificationPublisher = publisher
	}
}

func NewService(ticketRepository TicketRepository, options ...ServiceOption) *Service {
	service := &Service{ticketRepository: ticketRepository}
	for _, option := range options {
		option(service)
	}
	return service
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

	service.recordAuditEvent(ticket, actor.UserID, TicketAuditEventTypeCreated, "工单已创建", "", "")
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

func (service *Service) AddTicketComment(actor identity.IdentityContext, input AddTicketCommentInput) (TicketComment, error) {
	if service.commentRepository == nil {
		return TicketComment{}, ErrTicketCollaborationDisabled
	}

	ticket, err := service.GetTicket(actor, strings.TrimSpace(input.TicketID))
	if err != nil {
		return TicketComment{}, err
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return TicketComment{}, fmt.Errorf("%w: comment content is required", ErrInvalidTicketInput)
	}

	commentType := input.Type
	if commentType == "" {
		commentType = TicketCommentTypeComment
	}
	if commentType != TicketCommentTypeComment && commentType != TicketCommentTypeInternalNote {
		return TicketComment{}, fmt.Errorf("%w: unsupported comment type", ErrInvalidTicketInput)
	}
	if commentType == TicketCommentTypeInternalNote && !canManageTicket(actor, ticket) {
		// 内部备注只允许拥有工单写权限的处理方写入，避免终端用户把“内部协作区”当成公开对话区。
		return TicketComment{}, ErrTicketForbidden
	}

	now := time.Now()
	comment := service.commentRepository.Save(TicketComment{
		TicketID:       ticket.ID,
		OrganizationID: ticket.OrganizationID,
		AuthorID:       actor.UserID,
		Type:           commentType,
		Content:        content,
		CreatedAt:      now,
	})

	return comment, nil
}

func (service *Service) ListTicketTimeline(actor identity.IdentityContext, ticketID string) ([]TicketTimelineItem, error) {
	if service.commentRepository == nil || service.auditRepository == nil {
		return nil, ErrTicketCollaborationDisabled
	}

	ticket, err := service.GetTicket(actor, ticketID)
	if err != nil {
		return nil, err
	}

	items := make([]TicketTimelineItem, 0)
	for _, event := range service.auditRepository.ListByTicket(ticket.ID) {
		items = append(items, TicketTimelineItem{
			ID:             event.ID,
			TicketID:       event.TicketID,
			OrganizationID: event.OrganizationID,
			ActorID:        event.ActorID,
			ItemType:       TicketTimelineItemTypeAuditEvent,
			AuditEventType: event.Type,
			Content:        event.Content,
			FromValue:      event.FromValue,
			ToValue:        event.ToValue,
			CreatedAt:      event.CreatedAt,
		})
	}
	for _, comment := range service.commentRepository.ListByTicket(ticket.ID) {
		if comment.Type == TicketCommentTypeInternalNote && !canManageTicket(actor, ticket) {
			continue
		}
		items = append(items, TicketTimelineItem{
			ID:             comment.ID,
			TicketID:       comment.TicketID,
			OrganizationID: comment.OrganizationID,
			ActorID:        comment.AuthorID,
			ItemType:       TicketTimelineItemTypeComment,
			CommentType:    comment.Type,
			Content:        comment.Content,
			CreatedAt:      comment.CreatedAt,
		})
	}

	sort.Slice(items, func(left, right int) bool {
		if items[left].CreatedAt.Equal(items[right].CreatedAt) {
			if items[left].ItemType == items[right].ItemType {
				return items[left].ID < items[right].ID
			}
			return items[left].ItemType < items[right].ItemType
		}
		return items[left].CreatedAt.Before(items[right].CreatedAt)
	})

	return items, nil
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

	fromAssigneeID := ticket.AssigneeID
	ticket.AssigneeID = strings.TrimSpace(input.AssigneeID)
	ticket.UpdatedAt = time.Now()
	ticket = service.ticketRepository.Save(ticket)
	service.recordAuditEvent(ticket, actor.UserID, TicketAuditEventTypeAssignmentChanged, "工单指派已更新", fromAssigneeID, ticket.AssigneeID)
	service.publishAssignmentNotification(ticket, actor.UserID, fromAssigneeID)
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

	fromStatus := ticket.Status
	ticket.Status = input.ToStatus
	ticket.UpdatedAt = time.Now()
	ticket = service.ticketRepository.Save(ticket)
	service.recordAuditEvent(ticket, actor.UserID, TicketAuditEventTypeStatusChanged, "工单状态已更新", string(fromStatus), string(ticket.Status))
	service.publishStatusChangeNotification(ticket, actor.UserID, fromStatus)
	return ticket, nil
}

func (service *Service) publishAssignmentNotification(ticket Ticket, actorID string, fromAssigneeID string) {
	if service.notificationPublisher == nil {
		return
	}
	// 通知是工单主流程的旁路能力：这里即便渠道失败，也不能把已经成功的分派操作回滚掉。
	_ = service.notificationPublisher.Publish(context.Background(), notify.Event{
		Type:             notify.EventTypeTicketAssigned,
		OrganizationID:   ticket.OrganizationID,
		TicketID:         ticket.ID,
		ActorID:          actorID,
		RecipientUserIDs: uniqueRecipientUserIDs(ticket.RequesterID, ticket.AssigneeID),
		Subject:          "工单已分派",
		Content:          fmt.Sprintf("工单《%s》已分派给 %s。", ticket.Title, ticket.AssigneeID),
		Metadata: map[string]string{
			"from_assignee_id": strings.TrimSpace(fromAssigneeID),
			"to_assignee_id":   strings.TrimSpace(ticket.AssigneeID),
		},
		CreatedAt: time.Now(),
	})
}

func (service *Service) publishStatusChangeNotification(ticket Ticket, actorID string, fromStatus TicketStatus) {
	if service.notificationPublisher == nil {
		return
	}
	// 状态通知用于触达 requester / assignee，但其失败不应影响工单状态已经成功流转这一事实。
	_ = service.notificationPublisher.Publish(context.Background(), notify.Event{
		Type:             notify.EventTypeTicketStatusChanged,
		OrganizationID:   ticket.OrganizationID,
		TicketID:         ticket.ID,
		ActorID:          actorID,
		RecipientUserIDs: uniqueRecipientUserIDs(ticket.RequesterID, ticket.AssigneeID),
		Subject:          "工单状态已更新",
		Content:          fmt.Sprintf("工单《%s》状态已从 %s 变更为 %s。", ticket.Title, fromStatus, ticket.Status),
		Metadata: map[string]string{
			"from_status": string(fromStatus),
			"to_status":   string(ticket.Status),
		},
		CreatedAt: time.Now(),
	})
}

func uniqueRecipientUserIDs(userIDs ...string) []string {
	seen := make(map[string]struct{}, len(userIDs))
	recipients := make([]string, 0, len(userIDs))
	for _, rawUserID := range userIDs {
		userID := strings.TrimSpace(rawUserID)
		if userID == "" {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		recipients = append(recipients, userID)
	}
	return recipients
}

func (service *Service) recordAuditEvent(ticket Ticket, actorID string, eventType TicketAuditEventType, content string, fromValue string, toValue string) {
	if service.auditRepository == nil {
		return
	}

	// 审计事件在写操作发生时立即落地，这样谁做了什么、从什么值变成什么值可以被稳定追踪，而不是依赖查询时再回推。
	service.auditRepository.Save(TicketAuditEvent{
		TicketID:       ticket.ID,
		OrganizationID: ticket.OrganizationID,
		ActorID:        actorID,
		Type:           eventType,
		Content:        content,
		FromValue:      fromValue,
		ToValue:        toValue,
		CreatedAt:      time.Now(),
	})
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

	// 指派、状态流转和内部备注统一收敛为“工单写权限”，先把协作边界稳定下来。
	return actor.Permissions.Contains(identity.PermissionTicketWrite)
}

package ticket

import (
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

func TestAddTicketCommentAndListTimeline(t *testing.T) {
	service := newTicketCollaborationService()
	requester := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{
		Title:       "VPN 无法连接",
		Description: "今天上午开始无法连接公司 VPN",
		Category:    "network",
		Priority:    TicketPriorityHigh,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	comment, err := service.AddTicketComment(agent, AddTicketCommentInput{
		TicketID: createdTicket.ID,
		Type:     TicketCommentTypeComment,
		Content:  "请先确认 VPN 客户端是否在线。",
	})
	if err != nil {
		t.Fatalf("add ticket comment: %v", err)
	}
	if comment.ID == "" {
		t.Fatalf("expected comment id to be generated")
	}

	timeline, err := service.ListTicketTimeline(agent, createdTicket.ID)
	if err != nil {
		t.Fatalf("list ticket timeline: %v", err)
	}
	if len(timeline) < 2 {
		t.Fatalf("expected timeline to contain create event and comment, got %d items", len(timeline))
	}
	if timeline[len(timeline)-1].ItemType != TicketTimelineItemTypeComment {
		t.Fatalf("expected last timeline item to be comment, got %q", timeline[len(timeline)-1].ItemType)
	}
	if timeline[len(timeline)-1].Content != "请先确认 VPN 客户端是否在线。" {
		t.Fatalf("unexpected timeline comment content: %q", timeline[len(timeline)-1].Content)
	}
}

func TestEndUserCannotAddInternalNote(t *testing.T) {
	service := newTicketCollaborationService()
	requester := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{Title: "共享盘无权限", Description: "无法访问共享盘", Category: "storage", Priority: TicketPriorityHigh})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	_, err = service.AddTicketComment(requester, AddTicketCommentInput{
		TicketID: createdTicket.ID,
		Type:     TicketCommentTypeInternalNote,
		Content:  "这是一条内部备注。",
	})
	if err != ErrTicketForbidden {
		t.Fatalf("expected ticket forbidden, got %v", err)
	}
}

func TestAssignTicketRecordsAuditEvent(t *testing.T) {
	service := newTicketCollaborationService()
	requester := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{Title: "账号锁定", Description: "被锁定", Category: "account", Priority: TicketPriorityUrgent})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err := service.AssignTicket(agent, AssignTicketInput{TicketID: createdTicket.ID, AssigneeID: "user-agent-2"}); err != nil {
		t.Fatalf("assign ticket: %v", err)
	}

	timeline, err := service.ListTicketTimeline(agent, createdTicket.ID)
	if err != nil {
		t.Fatalf("list timeline: %v", err)
	}

	found := false
	for _, item := range timeline {
		if item.ItemType == TicketTimelineItemTypeAuditEvent && item.AuditEventType == TicketAuditEventTypeAssignmentChanged {
			found = true
			if item.FromValue != "" || item.ToValue != "user-agent-2" {
				t.Fatalf("unexpected assignment change values: from=%q to=%q", item.FromValue, item.ToValue)
			}
		}
	}
	if !found {
		t.Fatalf("expected assignment audit event to be recorded")
	}
}

func TestTransitionTicketStatusRecordsAuditEvent(t *testing.T) {
	service := newTicketCollaborationService()
	requester := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{Title: "邮箱异常", Description: "收不到邮件", Category: "mail", Priority: TicketPriorityMedium})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := service.TransitionTicketStatus(agent, TransitionTicketStatusInput{TicketID: createdTicket.ID, ToStatus: TicketStatusInProgress}); err != nil {
		t.Fatalf("transition ticket status: %v", err)
	}

	timeline, err := service.ListTicketTimeline(agent, createdTicket.ID)
	if err != nil {
		t.Fatalf("list timeline: %v", err)
	}

	found := false
	for _, item := range timeline {
		if item.ItemType == TicketTimelineItemTypeAuditEvent && item.AuditEventType == TicketAuditEventTypeStatusChanged {
			found = true
			if item.FromValue != string(TicketStatusOpen) || item.ToValue != string(TicketStatusInProgress) {
				t.Fatalf("unexpected status change values: from=%q to=%q", item.FromValue, item.ToValue)
			}
		}
	}
	if !found {
		t.Fatalf("expected status audit event to be recorded")
	}
}

func TestEndUserTimelineHidesInternalNotes(t *testing.T) {
	service := newTicketCollaborationService()
	requester := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{Title: "VPN 无法连接", Description: "今天上午开始无法连接", Category: "network", Priority: TicketPriorityHigh})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := service.AddTicketComment(agent, AddTicketCommentInput{TicketID: createdTicket.ID, Type: TicketCommentTypeInternalNote, Content: "需要排查证书是否过期。"}); err != nil {
		t.Fatalf("add internal note: %v", err)
	}
	if _, err := service.AddTicketComment(agent, AddTicketCommentInput{TicketID: createdTicket.ID, Type: TicketCommentTypeComment, Content: "请先重置客户端。"}); err != nil {
		t.Fatalf("add public comment: %v", err)
	}

	requesterTimeline, err := service.ListTicketTimeline(requester, createdTicket.ID)
	if err != nil {
		t.Fatalf("list requester timeline: %v", err)
	}
	for _, item := range requesterTimeline {
		if item.CommentType == TicketCommentTypeInternalNote {
			t.Fatalf("expected internal note to be hidden from requester")
		}
	}
}

func newTicketCollaborationService() *Service {
	return NewService(
		NewMemoryTicketRepository(),
		WithCollaborationDependencies(CollaborationDependencies{
			CommentRepository: NewMemoryTicketCommentRepository(),
			AuditRepository:   NewMemoryTicketAuditEventRepository(),
		}),
	)
}

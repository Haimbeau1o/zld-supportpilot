package ticket

import (
	"context"
	"errors"
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/notify"
)

type stubNotificationPublisher struct {
	events []notify.Event
	err    error
}

func (publisher *stubNotificationPublisher) Publish(_ context.Context, event notify.Event) error {
	publisher.events = append(publisher.events, event)
	return publisher.err
}

func TestAssignTicketPublishesNotification(t *testing.T) {
	publisher := &stubNotificationPublisher{}
	service := NewService(NewMemoryTicketRepository(), WithNotificationPublisher(publisher))
	requester := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{Title: "账号锁定", Description: "登录失败", Category: "account", Priority: TicketPriorityUrgent})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	updatedTicket, err := service.AssignTicket(agent, AssignTicketInput{TicketID: createdTicket.ID, AssigneeID: "user-agent-2"})
	if err != nil {
		t.Fatalf("assign ticket: %v", err)
	}
	if updatedTicket.AssigneeID != "user-agent-2" {
		t.Fatalf("expected assignee updated, got %q", updatedTicket.AssigneeID)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected 1 notification event, got %d", len(publisher.events))
	}
	if publisher.events[0].Type != notify.EventTypeTicketAssigned {
		t.Fatalf("expected event type %q, got %q", notify.EventTypeTicketAssigned, publisher.events[0].Type)
	}
	if len(publisher.events[0].RecipientUserIDs) != 2 {
		t.Fatalf("expected requester and assignee to be notified, got %+v", publisher.events[0].RecipientUserIDs)
	}
}

func TestAssignTicketIgnoresNotificationFailure(t *testing.T) {
	publisher := &stubNotificationPublisher{err: errors.New("webhook timeout")}
	service := NewService(NewMemoryTicketRepository(), WithNotificationPublisher(publisher))
	requester := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{Title: "账号锁定", Description: "登录失败", Category: "account", Priority: TicketPriorityUrgent})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	updatedTicket, err := service.AssignTicket(agent, AssignTicketInput{TicketID: createdTicket.ID, AssigneeID: "user-agent-2"})
	if err != nil {
		t.Fatalf("expected assignment success even if notification fails, got %v", err)
	}
	if updatedTicket.AssigneeID != "user-agent-2" {
		t.Fatalf("expected assignee updated, got %q", updatedTicket.AssigneeID)
	}
}

func TestTransitionTicketStatusPublishesNotification(t *testing.T) {
	publisher := &stubNotificationPublisher{}
	service := NewService(NewMemoryTicketRepository(), WithNotificationPublisher(publisher))
	requester := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{Title: "共享盘无权限", Description: "无法访问共享盘", Category: "storage", Priority: TicketPriorityHigh})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	_, err = service.AssignTicket(agent, AssignTicketInput{TicketID: createdTicket.ID, AssigneeID: agent.UserID})
	if err != nil {
		t.Fatalf("assign ticket: %v", err)
	}
	publisher.events = nil

	updatedTicket, err := service.TransitionTicketStatus(agent, TransitionTicketStatusInput{TicketID: createdTicket.ID, ToStatus: TicketStatusInProgress})
	if err != nil {
		t.Fatalf("transition ticket status: %v", err)
	}
	if updatedTicket.Status != TicketStatusInProgress {
		t.Fatalf("expected status in_progress, got %q", updatedTicket.Status)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected 1 notification event, got %d", len(publisher.events))
	}
	if publisher.events[0].Type != notify.EventTypeTicketStatusChanged {
		t.Fatalf("expected event type %q, got %q", notify.EventTypeTicketStatusChanged, publisher.events[0].Type)
	}
}

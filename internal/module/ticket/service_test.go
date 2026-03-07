package ticket

import (
	"errors"
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

func TestCreateTicket(t *testing.T) {
	service := NewService(NewMemoryTicketRepository())
	requester := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{
		Title:       "VPN 无法连接",
		Description: "今天上午开始无法连接公司 VPN",
		Category:    "network",
		Priority:    TicketPriorityHigh,
	})
	if err != nil {
		t.Fatalf("expected create ticket success, got error: %v", err)
	}

	if createdTicket.ID == "" {
		t.Fatalf("expected ticket id to be generated")
	}

	if createdTicket.OrganizationID != requester.OrganizationID {
		t.Fatalf("expected organization id %q, got %q", requester.OrganizationID, createdTicket.OrganizationID)
	}

	if createdTicket.RequesterID != requester.UserID {
		t.Fatalf("expected requester id %q, got %q", requester.UserID, createdTicket.RequesterID)
	}

	if createdTicket.Status != TicketStatusOpen {
		t.Fatalf("expected initial ticket status %q, got %q", TicketStatusOpen, createdTicket.Status)
	}
}

func TestListTickets(t *testing.T) {
	service := NewService(NewMemoryTicketRepository())
	endUser1 := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	endUser2 := identity.IdentityContext{
		UserID:         "user-end-2",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}
	otherOrgUser := identity.IdentityContext{
		UserID:         "user-other-1",
		OrganizationID: "org-2",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}

	if _, err := service.CreateTicket(endUser1, CreateTicketInput{Title: "电脑蓝屏", Description: "重启后仍失败", Category: "device", Priority: TicketPriorityHigh}); err != nil {
		t.Fatalf("create first ticket: %v", err)
	}
	if _, err := service.CreateTicket(endUser2, CreateTicketInput{Title: "邮箱异常", Description: "收不到邮件", Category: "mail", Priority: TicketPriorityMedium}); err != nil {
		t.Fatalf("create second ticket: %v", err)
	}
	if _, err := service.CreateTicket(otherOrgUser, CreateTicketInput{Title: "打印机异常", Description: "无法打印", Category: "office", Priority: TicketPriorityLow}); err != nil {
		t.Fatalf("create third ticket: %v", err)
	}

	selfTickets, err := service.ListTickets(endUser1, ListTicketsInput{})
	if err != nil {
		t.Fatalf("expected end user list success, got error: %v", err)
	}
	if len(selfTickets) != 1 {
		t.Fatalf("expected end user to see 1 ticket, got %d", len(selfTickets))
	}
	if selfTickets[0].RequesterID != endUser1.UserID {
		t.Fatalf("expected end user to see only own ticket")
	}

	tenantTickets, err := service.ListTickets(agent, ListTicketsInput{})
	if err != nil {
		t.Fatalf("expected agent list success, got error: %v", err)
	}
	if len(tenantTickets) != 2 {
		t.Fatalf("expected agent to see 2 tenant tickets, got %d", len(tenantTickets))
	}
}

func TestAssignTicket(t *testing.T) {
	service := NewService(NewMemoryTicketRepository())
	requester := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{
		Title:       "账号锁定",
		Description: "登录多次失败后被锁定",
		Category:    "account",
		Priority:    TicketPriorityUrgent,
	})
	if err != nil {
		t.Fatalf("expected create ticket success, got error: %v", err)
	}

	updatedTicket, err := service.AssignTicket(agent, AssignTicketInput{
		TicketID:   createdTicket.ID,
		AssigneeID: "user-agent-2",
	})
	if err != nil {
		t.Fatalf("expected assign ticket success, got error: %v", err)
	}

	if updatedTicket.AssigneeID != "user-agent-2" {
		t.Fatalf("expected assignee user-agent-2, got %q", updatedTicket.AssigneeID)
	}
}

func TestTransitionTicketStatus(t *testing.T) {
	service := NewService(NewMemoryTicketRepository())
	requester := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	createdTicket, err := service.CreateTicket(requester, CreateTicketInput{
		Title:       "共享盘无权限",
		Description: "无法访问部门共享盘",
		Category:    "storage",
		Priority:    TicketPriorityHigh,
	})
	if err != nil {
		t.Fatalf("expected create ticket success, got error: %v", err)
	}

	_, err = service.TransitionTicketStatus(agent, TransitionTicketStatusInput{
		TicketID: createdTicket.ID,
		ToStatus: TicketStatusResolved,
	})
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected invalid status transition error, got %v", err)
	}
}

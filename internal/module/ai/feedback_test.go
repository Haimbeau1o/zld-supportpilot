package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

func TestSubmitAnswerFeedbackResolvedDoesNotCreateTicket(t *testing.T) {
	endUser := identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	intakeRepository := NewMemoryUnifiedIntakeRepository()
	feedbackRepository := NewMemoryAnswerFeedbackRepository()
	workspace := &feedbackTicketWorkspace{}
	service := NewService(ServiceDependencies{
		TicketWorkspace:    workspace,
		IntakeRepository:   intakeRepository,
		FeedbackRepository: feedbackRepository,
	})

	intake := intakeRepository.Save(UnifiedIntakeRecord{
		OrganizationID:  endUser.OrganizationID,
		RequesterID:     endUser.UserID,
		KnowledgeBaseID: "kb-1",
		Question:        "VPN 无法连接怎么办？",
		ResultType:      UnifiedIntakeResultTypeAnswered,
		AnswerStatus:    AnswerStatusAnswered,
		Answer:          "请先重置 VPN 客户端。",
		Confidence:      0.82,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	result, err := service.SubmitAnswerFeedback(context.Background(), endUser, AnswerFeedbackInput{
		IntakeID: intake.ID,
		Status:   AnswerFeedbackStatusResolved,
		Comment:  "按步骤操作后已恢复",
	})
	if err != nil {
		t.Fatalf("submit answer feedback: %v", err)
	}
	if result.TicketAction != AnswerFeedbackTicketActionNone {
		t.Fatalf("expected no ticket action, got %q", result.TicketAction)
	}
	if result.TicketID != "" {
		t.Fatalf("expected empty ticket id, got %q", result.TicketID)
	}
	if workspace.createTicketCalls != 0 {
		t.Fatalf("expected no ticket creation, got %d calls", workspace.createTicketCalls)
	}
}

func TestSubmitAnswerFeedbackUnresolvedCreatesTicket(t *testing.T) {
	endUser := identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	intakeRepository := NewMemoryUnifiedIntakeRepository()
	feedbackRepository := NewMemoryAnswerFeedbackRepository()
	workspace := &feedbackTicketWorkspace{
		createdTicket: ticket.Ticket{ID: "ticket-2", OrganizationID: "org-1", RequesterID: "user-1", Title: "VPN 无法连接怎么办？", Status: ticket.TicketStatusOpen},
	}
	service := NewService(ServiceDependencies{
		TicketWorkspace:    workspace,
		IntakeRepository:   intakeRepository,
		FeedbackRepository: feedbackRepository,
	})

	intake := intakeRepository.Save(UnifiedIntakeRecord{
		OrganizationID:  endUser.OrganizationID,
		RequesterID:     endUser.UserID,
		KnowledgeBaseID: "kb-1",
		Question:        "VPN 无法连接怎么办？",
		ResultType:      UnifiedIntakeResultTypeAnswered,
		AnswerStatus:    AnswerStatusAnswered,
		Answer:          "请先重置 VPN 客户端。",
		Confidence:      0.82,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	result, err := service.SubmitAnswerFeedback(context.Background(), endUser, AnswerFeedbackInput{
		IntakeID: intake.ID,
		Status:   AnswerFeedbackStatusUnresolved,
		Comment:  "按步骤执行后仍然失败",
	})
	if err != nil {
		t.Fatalf("submit answer feedback: %v", err)
	}
	if result.TicketAction != AnswerFeedbackTicketActionTicketCreated {
		t.Fatalf("expected ticket created action, got %q", result.TicketAction)
	}
	if result.TicketID != "ticket-2" {
		t.Fatalf("expected ticket id %q, got %q", "ticket-2", result.TicketID)
	}
	if workspace.createTicketCalls != 1 {
		t.Fatalf("expected 1 ticket creation, got %d", workspace.createTicketCalls)
	}
	if workspace.createdTicketInput.Description == "" {
		t.Fatalf("expected escalation description to be generated")
	}
}

func TestSubmitAnswerFeedbackDoesNotCreateDuplicateTicket(t *testing.T) {
	endUser := identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	intakeRepository := NewMemoryUnifiedIntakeRepository()
	feedbackRepository := NewMemoryAnswerFeedbackRepository()
	workspace := &feedbackTicketWorkspace{
		createdTicket: ticket.Ticket{ID: "ticket-2", OrganizationID: "org-1", RequesterID: "user-1", Title: "VPN 无法连接怎么办？", Status: ticket.TicketStatusOpen},
	}
	service := NewService(ServiceDependencies{
		TicketWorkspace:    workspace,
		IntakeRepository:   intakeRepository,
		FeedbackRepository: feedbackRepository,
	})

	intake := intakeRepository.Save(UnifiedIntakeRecord{
		OrganizationID:    endUser.OrganizationID,
		RequesterID:       endUser.UserID,
		KnowledgeBaseID:   "kb-1",
		Question:          "VPN 无法连接怎么办？",
		ResultType:        UnifiedIntakeResultTypeAnswered,
		AnswerStatus:      AnswerStatusAnswered,
		Answer:            "请先重置 VPN 客户端。",
		Confidence:        0.82,
		EscalatedTicketID: "ticket-9",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	})

	result, err := service.SubmitAnswerFeedback(context.Background(), endUser, AnswerFeedbackInput{
		IntakeID: intake.ID,
		Status:   AnswerFeedbackStatusUnresolved,
	})
	if err != nil {
		t.Fatalf("submit answer feedback: %v", err)
	}
	if result.TicketAction != AnswerFeedbackTicketActionTicketExisting {
		t.Fatalf("expected existing ticket action, got %q", result.TicketAction)
	}
	if result.TicketID != "ticket-9" {
		t.Fatalf("expected ticket id %q, got %q", "ticket-9", result.TicketID)
	}
	if workspace.createTicketCalls != 0 {
		t.Fatalf("expected no duplicate ticket creation, got %d", workspace.createTicketCalls)
	}
}

func TestGetAnswerFeedbackStatsRejectsEndUser(t *testing.T) {
	endUser := identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	service := NewService(ServiceDependencies{
		TicketWorkspace:    &feedbackTicketWorkspace{},
		IntakeRepository:   NewMemoryUnifiedIntakeRepository(),
		FeedbackRepository: NewMemoryAnswerFeedbackRepository(),
	})

	_, err := service.GetAnswerFeedbackStats(context.Background(), endUser)
	if !errors.Is(err, ticket.ErrTicketForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestGetAnswerFeedbackStatsCountsLatestFeedbacks(t *testing.T) {
	agent := identity.IdentityContext{
		UserID:         "agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}
	requester := identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	intakeRepository := NewMemoryUnifiedIntakeRepository()
	feedbackRepository := NewMemoryAnswerFeedbackRepository()
	workspace := &feedbackTicketWorkspace{createdTicket: ticket.Ticket{ID: "ticket-7", OrganizationID: "org-1", RequesterID: "user-1", Title: "VPN", Status: ticket.TicketStatusOpen}}
	service := NewService(ServiceDependencies{
		TicketWorkspace:    workspace,
		IntakeRepository:   intakeRepository,
		FeedbackRepository: feedbackRepository,
	})

	intakeA := intakeRepository.Save(UnifiedIntakeRecord{
		OrganizationID:  requester.OrganizationID,
		RequesterID:     requester.UserID,
		KnowledgeBaseID: "kb-1",
		Question:        "VPN 无法连接怎么办？",
		ResultType:      UnifiedIntakeResultTypeAnswered,
		AnswerStatus:    AnswerStatusAnswered,
		Answer:          "请先重置 VPN 客户端。",
		Confidence:      0.82,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	intakeB := intakeRepository.Save(UnifiedIntakeRecord{
		OrganizationID:  requester.OrganizationID,
		RequesterID:     requester.UserID,
		KnowledgeBaseID: "kb-1",
		Question:        "邮箱发不出去怎么办？",
		ResultType:      UnifiedIntakeResultTypeAnswered,
		AnswerStatus:    AnswerStatusAnswered,
		Answer:          "请检查 SMTP 配置。",
		Confidence:      0.76,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	if _, err := service.SubmitAnswerFeedback(context.Background(), requester, AnswerFeedbackInput{IntakeID: intakeA.ID, Status: AnswerFeedbackStatusResolved}); err != nil {
		t.Fatalf("submit resolved feedback: %v", err)
	}
	if _, err := service.SubmitAnswerFeedback(context.Background(), requester, AnswerFeedbackInput{IntakeID: intakeA.ID, Status: AnswerFeedbackStatusInaccurate}); err != nil {
		t.Fatalf("submit inaccurate feedback: %v", err)
	}
	if _, err := service.SubmitAnswerFeedback(context.Background(), requester, AnswerFeedbackInput{IntakeID: intakeB.ID, Status: AnswerFeedbackStatusUnresolved}); err != nil {
		t.Fatalf("submit unresolved feedback: %v", err)
	}

	stats, err := service.GetAnswerFeedbackStats(context.Background(), agent)
	if err != nil {
		t.Fatalf("get answer feedback stats: %v", err)
	}
	if stats.Total != 2 {
		t.Fatalf("expected total %d, got %d", 2, stats.Total)
	}
	if stats.Resolved != 0 {
		t.Fatalf("expected resolved %d, got %d", 0, stats.Resolved)
	}
	if stats.Inaccurate != 1 {
		t.Fatalf("expected inaccurate %d, got %d", 1, stats.Inaccurate)
	}
	if stats.Unresolved != 1 {
		t.Fatalf("expected unresolved %d, got %d", 1, stats.Unresolved)
	}
}

type feedbackTicketWorkspace struct {
	createdTicket      ticket.Ticket
	createdTicketInput ticket.CreateTicketInput
	createTicketCalls  int
}

func (workspace *feedbackTicketWorkspace) GetTicket(identity.IdentityContext, string) (ticket.Ticket, error) {
	return ticket.Ticket{}, nil
}

func (workspace *feedbackTicketWorkspace) ListTicketTimeline(identity.IdentityContext, string) ([]ticket.TicketTimelineItem, error) {
	return nil, nil
}

func (workspace *feedbackTicketWorkspace) AddTicketComment(identity.IdentityContext, ticket.AddTicketCommentInput) (ticket.TicketComment, error) {
	return ticket.TicketComment{}, nil
}

func (workspace *feedbackTicketWorkspace) CreateTicket(actor identity.IdentityContext, input ticket.CreateTicketInput) (ticket.Ticket, error) {
	workspace.createTicketCalls++
	workspace.createdTicketInput = input
	if workspace.createdTicket.ID != "" {
		return workspace.createdTicket, nil
	}
	return ticket.Ticket{ID: "ticket-created", OrganizationID: actor.OrganizationID, RequesterID: actor.UserID, Title: input.Title, Description: input.Description, Category: input.Category, Priority: input.Priority, Status: ticket.TicketStatusOpen}, nil
}

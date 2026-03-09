package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/notify"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

type stubAINotificationPublisher struct {
	events []notify.Event
	err    error
}

func (publisher *stubAINotificationPublisher) Publish(_ context.Context, event notify.Event) error {
	publisher.events = append(publisher.events, event)
	return publisher.err
}

func TestHandleUnifiedIntakePublishesEscalationNotificationWhenKnowledgeIsDegraded(t *testing.T) {
	endUser := identity.IdentityContext{UserID: "user-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	workspace := &stubTicketWorkspace{createdTicket: ticket.Ticket{ID: "ticket-1", OrganizationID: "org-1", RequesterID: "user-1", Title: "VPN 无法连接怎么办？", Status: ticket.TicketStatusOpen}}
	publisher := &stubAINotificationPublisher{}
	service := NewService(ServiceDependencies{
		TicketWorkspace:       workspace,
		IntakeRepository:      NewMemoryUnifiedIntakeRepository(),
		FeedbackRepository:    NewMemoryAnswerFeedbackRepository(),
		NotificationPublisher: publisher,
		KnowledgeAnswerer: stubKnowledgeAnswerer{result: AnswerResult{
			Status:     AnswerStatusDegraded,
			Answer:     degradedAnswerResult().Answer,
			Confidence: 0,
		}},
	})

	result, err := service.HandleUnifiedIntake(context.Background(), endUser, UnifiedIntakeInput{KnowledgeBaseID: "kb-1", Question: "VPN 无法连接怎么办？", TopK: 3})
	if err != nil {
		t.Fatalf("handle unified intake: %v", err)
	}
	if result.TicketID != "ticket-1" {
		t.Fatalf("expected created ticket id %q, got %q", "ticket-1", result.TicketID)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected 1 escalation notification, got %d", len(publisher.events))
	}
	if publisher.events[0].Type != notify.EventTypeAIEscalated {
		t.Fatalf("expected event type %q, got %q", notify.EventTypeAIEscalated, publisher.events[0].Type)
	}
}

func TestSubmitAnswerFeedbackPublishesEscalationNotificationWhenTicketCreated(t *testing.T) {
	endUser := identity.IdentityContext{UserID: "user-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	intakeRepository := NewMemoryUnifiedIntakeRepository()
	feedbackRepository := NewMemoryAnswerFeedbackRepository()
	workspace := &feedbackTicketWorkspace{createdTicket: ticket.Ticket{ID: "ticket-2", OrganizationID: "org-1", RequesterID: "user-1", Title: "VPN 无法连接怎么办？", Status: ticket.TicketStatusOpen}}
	publisher := &stubAINotificationPublisher{}
	service := NewService(ServiceDependencies{TicketWorkspace: workspace, IntakeRepository: intakeRepository, FeedbackRepository: feedbackRepository, NotificationPublisher: publisher})

	intake := intakeRepository.Save(UnifiedIntakeRecord{OrganizationID: endUser.OrganizationID, RequesterID: endUser.UserID, KnowledgeBaseID: "kb-1", Question: "VPN 无法连接怎么办？", ResultType: UnifiedIntakeResultTypeAnswered, AnswerStatus: AnswerStatusAnswered, Answer: "请先重置 VPN 客户端。", Confidence: 0.82, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	result, err := service.SubmitAnswerFeedback(context.Background(), endUser, AnswerFeedbackInput{IntakeID: intake.ID, Status: AnswerFeedbackStatusUnresolved, Comment: "仍未解决"})
	if err != nil {
		t.Fatalf("submit answer feedback: %v", err)
	}
	if result.TicketID != "ticket-2" {
		t.Fatalf("expected ticket id %q, got %q", "ticket-2", result.TicketID)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected 1 escalation notification, got %d", len(publisher.events))
	}
	if publisher.events[0].Type != notify.EventTypeAIEscalated {
		t.Fatalf("expected event type %q, got %q", notify.EventTypeAIEscalated, publisher.events[0].Type)
	}
}

func TestSubmitAnswerFeedbackIgnoresNotificationFailure(t *testing.T) {
	endUser := identity.IdentityContext{UserID: "user-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	intakeRepository := NewMemoryUnifiedIntakeRepository()
	feedbackRepository := NewMemoryAnswerFeedbackRepository()
	workspace := &feedbackTicketWorkspace{createdTicket: ticket.Ticket{ID: "ticket-2", OrganizationID: "org-1", RequesterID: "user-1", Title: "VPN 无法连接怎么办？", Status: ticket.TicketStatusOpen}}
	publisher := &stubAINotificationPublisher{err: errors.New("email sender unavailable")}
	service := NewService(ServiceDependencies{TicketWorkspace: workspace, IntakeRepository: intakeRepository, FeedbackRepository: feedbackRepository, NotificationPublisher: publisher})

	intake := intakeRepository.Save(UnifiedIntakeRecord{OrganizationID: endUser.OrganizationID, RequesterID: endUser.UserID, KnowledgeBaseID: "kb-1", Question: "VPN 无法连接怎么办？", ResultType: UnifiedIntakeResultTypeAnswered, AnswerStatus: AnswerStatusAnswered, Answer: "请先重置 VPN 客户端。", Confidence: 0.82, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	result, err := service.SubmitAnswerFeedback(context.Background(), endUser, AnswerFeedbackInput{IntakeID: intake.ID, Status: AnswerFeedbackStatusUnresolved, Comment: "仍未解决"})
	if err != nil {
		t.Fatalf("expected feedback success even if notification fails, got %v", err)
	}
	if result.TicketID != "ticket-2" {
		t.Fatalf("expected ticket id %q, got %q", "ticket-2", result.TicketID)
	}
}

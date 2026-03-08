package ai

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

func TestGenerateTicketAssist(t *testing.T) {
	agent := identity.IdentityContext{
		UserID:         "agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}
	service := NewService(ServiceDependencies{
		TicketWorkspace: stubTicketWorkspace{
			ticket: ticket.Ticket{
				ID:             "ticket-1",
				OrganizationID: "org-1",
				RequesterID:    "user-1",
				Title:          "VPN 无法连接",
				Description:    "今天上午开始无法连接公司 VPN，重试后仍失败。",
				Category:       "network",
				Priority:       ticket.TicketPriorityHigh,
				Status:         ticket.TicketStatusOpen,
				CreatedAt:      time.Now().Add(-30 * time.Minute),
				UpdatedAt:      time.Now().Add(-5 * time.Minute),
			},
			timeline: []ticket.TicketTimelineItem{{
				ID:          "comment-1",
				TicketID:    "ticket-1",
				ActorID:     "user-1",
				ItemType:    ticket.TicketTimelineItemTypeComment,
				CommentType: ticket.TicketCommentTypeComment,
				Content:     "错误提示是 691，客户端已重装。",
				CreatedAt:   time.Now().Add(-3 * time.Minute),
			}},
			recordedComment: ticket.TicketComment{
				ID:        "comment-ai-1",
				TicketID:  "ticket-1",
				AuthorID:  "agent-1",
				Type:      ticket.TicketCommentTypeInternalNote,
				Content:   "【AI 执行记录】",
				CreatedAt: time.Now(),
			},
		},
		TicketAnalyzer: TemplateTicketAnalyzer{},
		KnowledgeAnswerer: stubKnowledgeAnswerer{result: AnswerResult{
			Status:     AnswerStatusAnswered,
			Answer:     "根据知识库资料，建议先重置 VPN 客户端配置，再检查账号状态。",
			Confidence: 0.88,
			Citations: []Citation{{
				ChunkID:    "chunk-1",
				DocumentID: "doc-1",
				Score:      0.88,
				Content:    "VPN 无法连接时，请先重置客户端配置。",
			}},
		}},
	})

	result, err := service.GenerateTicketAssist(context.Background(), agent, GenerateTicketAssistInput{
		TicketID:        "ticket-1",
		KnowledgeBaseID: "kb-1",
	})
	if err != nil {
		t.Fatalf("generate ticket assist: %v", err)
	}
	if result.CategorySuggestion == "" {
		t.Fatalf("expected category suggestion")
	}
	if result.Summary == "" {
		t.Fatalf("expected summary")
	}
	if result.ReplyDraft == "" {
		t.Fatalf("expected reply draft")
	}
	if result.RecordedCommentID != "comment-ai-1" {
		t.Fatalf("expected recorded comment id comment-ai-1, got %q", result.RecordedCommentID)
	}
	if len(result.Citations) == 0 {
		t.Fatalf("expected citations to be returned")
	}
}

func TestEndUserCannotGenerateTicketAssist(t *testing.T) {
	endUser := identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	service := NewService(ServiceDependencies{
		TicketWorkspace:   stubTicketWorkspace{},
		TicketAnalyzer:    TemplateTicketAnalyzer{},
		KnowledgeAnswerer: stubKnowledgeAnswerer{},
	})

	_, err := service.GenerateTicketAssist(context.Background(), endUser, GenerateTicketAssistInput{
		TicketID:        "ticket-1",
		KnowledgeBaseID: "kb-1",
	})
	if err == nil {
		t.Fatalf("expected forbidden error for end user")
	}
}

func TestGenerateTicketAssistDegradedReplyDraft(t *testing.T) {
	agent := identity.IdentityContext{
		UserID:         "agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}
	service := NewService(ServiceDependencies{
		TicketWorkspace: stubTicketWorkspace{
			ticket: ticket.Ticket{
				ID:             "ticket-1",
				OrganizationID: "org-1",
				RequesterID:    "user-1",
				Title:          "共享盘无权限",
				Description:    "用户无法访问部门共享盘。",
				Category:       "storage",
				Priority:       ticket.TicketPriorityMedium,
				Status:         ticket.TicketStatusOpen,
				CreatedAt:      time.Now().Add(-20 * time.Minute),
				UpdatedAt:      time.Now().Add(-5 * time.Minute),
			},
			recordedComment: ticket.TicketComment{ID: "comment-ai-2", TicketID: "ticket-1", AuthorID: "agent-1", Type: ticket.TicketCommentTypeInternalNote, Content: "【AI 执行记录】", CreatedAt: time.Now()},
		},
		TicketAnalyzer: TemplateTicketAnalyzer{},
		KnowledgeAnswerer: stubKnowledgeAnswerer{result: AnswerResult{
			Status:     AnswerStatusDegraded,
			Answer:     degradedAnswerResult().Answer,
			Confidence: 0,
			Citations:  []Citation{},
		}},
	})

	result, err := service.GenerateTicketAssist(context.Background(), agent, GenerateTicketAssistInput{
		TicketID:        "ticket-1",
		KnowledgeBaseID: "kb-1",
	})
	if err != nil {
		t.Fatalf("generate ticket assist: %v", err)
	}
	if !strings.Contains(result.ReplyDraft, "建议转人工") {
		t.Fatalf("expected degraded reply draft, got %q", result.ReplyDraft)
	}
	if result.Summary == "" {
		t.Fatalf("expected summary even when degraded")
	}
}

type stubKnowledgeAnswerer struct {
	result AnswerResult
	err    error
}

func (answerer stubKnowledgeAnswerer) AskKnowledgeQuestion(context.Context, identity.IdentityContext, AskKnowledgeQuestionInput) (AnswerResult, error) {
	if answerer.err != nil {
		return AnswerResult{}, answerer.err
	}
	return answerer.result, nil
}

type stubTicketWorkspace struct {
	ticket          ticket.Ticket
	timeline        []ticket.TicketTimelineItem
	recordedComment ticket.TicketComment
}

func (workspace stubTicketWorkspace) GetTicket(actor identity.IdentityContext, ticketID string) (ticket.Ticket, error) {
	return workspace.ticket, nil
}

func (workspace stubTicketWorkspace) ListTicketTimeline(actor identity.IdentityContext, ticketID string) ([]ticket.TicketTimelineItem, error) {
	return workspace.timeline, nil
}

func (workspace stubTicketWorkspace) AddTicketComment(actor identity.IdentityContext, input ticket.AddTicketCommentInput) (ticket.TicketComment, error) {
	return workspace.recordedComment, nil
}

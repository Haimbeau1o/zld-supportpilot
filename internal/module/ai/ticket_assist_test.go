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
		TicketWorkspace: &stubTicketWorkspace{
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
	if result.CategorySuggestion == "" || result.Summary == "" || result.ReplyDraft == "" {
		t.Fatalf("expected ticket assist content to be populated")
	}
	if result.RecordedCommentID != "comment-ai-1" {
		t.Fatalf("expected recorded comment id %q, got %q", "comment-ai-1", result.RecordedCommentID)
	}
}

func TestGenerateTicketAssistRejectsEndUser(t *testing.T) {
	endUser := identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	service := NewService(ServiceDependencies{
		TicketWorkspace:   &stubTicketWorkspace{},
		TicketAnalyzer:    TemplateTicketAnalyzer{},
		KnowledgeAnswerer: stubKnowledgeAnswerer{result: AnswerResult{Status: AnswerStatusAnswered, Answer: "ok", Confidence: 0.9}},
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
		TicketWorkspace: &stubTicketWorkspace{
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

func TestGenerateTicketAssistUsesLatestNonAIContext(t *testing.T) {
	agent := identity.IdentityContext{
		UserID:         "agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}
	capturedInput := &AskKnowledgeQuestionInput{}
	service := NewService(ServiceDependencies{
		TicketWorkspace: &stubTicketWorkspace{
			ticket: ticket.Ticket{
				ID:             "ticket-1",
				OrganizationID: "org-1",
				RequesterID:    "user-1",
				Title:          "VPN 无法连接",
				Description:    "今天上午开始无法连接公司 VPN。",
				Category:       "network",
				Priority:       ticket.TicketPriorityHigh,
				Status:         ticket.TicketStatusOpen,
				CreatedAt:      time.Now().Add(-30 * time.Minute),
				UpdatedAt:      time.Now().Add(-5 * time.Minute),
			},
			timeline: []ticket.TicketTimelineItem{
				{ID: "comment-1", TicketID: "ticket-1", ActorID: "user-1", ItemType: ticket.TicketTimelineItemTypeComment, CommentType: ticket.TicketCommentTypeComment, Content: "第一次补充：客户端已重装。", CreatedAt: time.Now().Add(-4 * time.Minute)},
				{ID: "comment-ai-0", TicketID: "ticket-1", ActorID: "agent-1", ItemType: ticket.TicketTimelineItemTypeComment, CommentType: ticket.TicketCommentTypeInternalNote, Content: "【AI 执行记录】\n上次建议：请检查网络。", CreatedAt: time.Now().Add(-3 * time.Minute)},
				{ID: "comment-2", TicketID: "ticket-1", ActorID: "user-1", ItemType: ticket.TicketTimelineItemTypeComment, CommentType: ticket.TicketCommentTypeComment, Content: "最新补充：错误码 691。", CreatedAt: time.Now().Add(-2 * time.Minute)},
			},
			recordedComment: ticket.TicketComment{ID: "comment-ai-1", TicketID: "ticket-1", AuthorID: "agent-1", Type: ticket.TicketCommentTypeInternalNote, Content: "【AI 执行记录】", CreatedAt: time.Now()},
		},
		TicketAnalyzer: TemplateTicketAnalyzer{},
		KnowledgeAnswerer: stubKnowledgeAnswerer{
			capturedInput: capturedInput,
			result:        AnswerResult{Status: AnswerStatusAnswered, Answer: "建议先重置 VPN 客户端配置。", Confidence: 0.8},
		},
	})

	_, err := service.GenerateTicketAssist(context.Background(), agent, GenerateTicketAssistInput{
		TicketID:        "ticket-1",
		KnowledgeBaseID: "kb-1",
	})
	if err != nil {
		t.Fatalf("generate ticket assist: %v", err)
	}
	if !strings.Contains(capturedInput.Question, "最新补充：错误码 691") {
		t.Fatalf("expected latest public comment in question context, got %q", capturedInput.Question)
	}
	if strings.Contains(capturedInput.Question, "第一次补充：客户端已重装") {
		t.Fatalf("expected old comment to be skipped, got %q", capturedInput.Question)
	}
	if strings.Contains(capturedInput.Question, "【AI 执行记录】") {
		t.Fatalf("expected AI execution note to be skipped, got %q", capturedInput.Question)
	}
}

type stubKnowledgeAnswerer struct {
	result        AnswerResult
	err           error
	capturedInput *AskKnowledgeQuestionInput
}

func (answerer stubKnowledgeAnswerer) AskKnowledgeQuestion(_ context.Context, _ identity.IdentityContext, input AskKnowledgeQuestionInput) (AnswerResult, error) {
	if answerer.capturedInput != nil {
		*answerer.capturedInput = input
	}
	if answerer.err != nil {
		return AnswerResult{}, answerer.err
	}
	return answerer.result, nil
}

type stubTicketWorkspace struct {
	ticket             ticket.Ticket
	timeline           []ticket.TicketTimelineItem
	recordedComment    ticket.TicketComment
	createdTicket      ticket.Ticket
	createdTicketInput ticket.CreateTicketInput
}

func (workspace *stubTicketWorkspace) GetTicket(actor identity.IdentityContext, ticketID string) (ticket.Ticket, error) {
	return workspace.ticket, nil
}

func (workspace *stubTicketWorkspace) ListTicketTimeline(actor identity.IdentityContext, ticketID string) ([]ticket.TicketTimelineItem, error) {
	return workspace.timeline, nil
}

func (workspace *stubTicketWorkspace) AddTicketComment(actor identity.IdentityContext, input ticket.AddTicketCommentInput) (ticket.TicketComment, error) {
	return workspace.recordedComment, nil
}

func (workspace *stubTicketWorkspace) CreateTicket(actor identity.IdentityContext, input ticket.CreateTicketInput) (ticket.Ticket, error) {
	workspace.createdTicketInput = input
	if workspace.createdTicket.ID != "" {
		return workspace.createdTicket, nil
	}
	return ticket.Ticket{ID: "ticket-created", OrganizationID: actor.OrganizationID, RequesterID: actor.UserID, Title: input.Title, Description: input.Description, Category: input.Category, Priority: input.Priority, Status: ticket.TicketStatusOpen}, nil
}
